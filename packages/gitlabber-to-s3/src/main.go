package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"path/filepath"

	// "github.com/aws/aws-sdk-go"
	git "github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/viper"
	"github.com/xanzy/go-gitlab"
)

type Project struct {
	Name              string `json:"name"`
	SSHURLToRepo      string `json:"ssh_url_to_repo"`
	PathWithNamespace string `json:"path_with_namespace"`
}

var s3Bucket string
var s3Prefix string
var s3Region string
var gitlabBaseURL string
var token string
var sshKeyPath string

func main() {
	viper.AutomaticEnv()
	homeDir, _ := os.UserHomeDir()
	sshKeyPath = filepath.Join(homeDir, ".ssh/id_rsa")
	// sshKeyPath = filepath.Join("id_rsa")
	// fmt.Printf("SSH Private Key Path: %s\n", sshKeyPath)
	viper.SetDefault("S3_PREFIX", "gitlab-backup")
	viper.SetDefault("S3_REGION", "eu-west-2")
	viper.SetDefault("GITLAB_BASE_URL", "https://gitlab.com/api/v4")

	s3Bucket = viper.GetString("S3_BUCKET")
	s3Prefix = viper.GetString("S3_PREFIX")
	s3Region = viper.GetString("S3_REGION")
	gitlabBaseURL = viper.GetString("GITLAB_BASE_URL")
	token = viper.GetString("GITLAB_TOKEN")

	if s3Bucket == "" {
		fmt.Println("No S3_BUCKET env var set")
		return
	}

	if token == "" {
		fmt.Println("Error: GITLAB_TOKEN environment variable is not set")
		return
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	git, err := gitlab.NewClient(token, gitlab.WithHTTPClient(httpClient), gitlab.WithBaseURL(fmt.Sprintf("%s/api/v4", gitlabBaseURL)))
	if err != nil {
		fmt.Println("Error creating GitLab client:", err)
		return
	}

	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 1000, // Adjust the number of projects per page
		},
	}

	var allProjects []*gitlab.Project
	for {
		projects, resp, err := git.Projects.ListProjects(opt)
		if err != nil {
			fmt.Println("Error fetching projects:", err)
			return
		}

		allProjects = append(allProjects, projects...)
		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		opt.Page = resp.NextPage
	}
	fmt.Printf("Found %d project(s)\n", len(allProjects))

	var wg sync.WaitGroup
	for _, project := range allProjects {
		wg.Add(1)
		go func(p *gitlab.Project) {
			defer wg.Done()
			cloneProject(Project{
				Name:              p.Name,
				SSHURLToRepo:      p.SSHURLToRepo,
				PathWithNamespace: p.PathWithNamespace,
			})
		}(project)
	}

	wg.Wait()
	fmt.Println("Cloning complete.")

	timestamp := time.Now().Format("2006-01-02T150405")
	archiveFile := fmt.Sprintf("./%s.tar.gz", timestamp)

	err = tarGzipFolder("./backup", archiveFile)
	if err != nil {
		log.Fatalf("Failed to create archive: %v", err)
	}
	fmt.Printf("Beginning upload...")
	err = uploadToS3(s3Bucket, s3Prefix, archiveFile)
	if err != nil {
		fmt.Printf("failed - %s\n", err)
		log.Fatalf("Failed to upload to S3: %v", err)
	}
	fmt.Println("done!")
}

func fetchProjects(token string, gitlabURL string) ([]Project, error) {
	req, err := http.NewRequest("GET", gitlabURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch projects: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func cloneProject(project Project) {
	const maxRetries = 5
	const retryDelay = 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := tryCloneProject(project); err != nil {
			// fmt.Println(project.SSHURLToRepo)
			// fmt.Printf("Attempt %d failed to clone project %s: %v\n", attempt, project.Name, err)
			if attempt < maxRetries {
				// fmt.Println("Retrying after delay...")
				time.Sleep(retryDelay)
			} else {
				fmt.Printf("FAILED CLONING: %s (attempt %d/%d) - %s\n", project.SSHURLToRepo, attempt, maxRetries, err)
				return
			}
		} else {
			fmt.Printf("SUCCESSFULLY CLONED: %s (attempt %d/%d)\n", project.SSHURLToRepo, attempt, maxRetries)
			return
		}
	}
}

func tryCloneProject(project Project) error {

	sshAuth, err := gitssh.NewPublicKeysFromFile("git", sshKeyPath, "")
	if err != nil {
		return err
	}
	sshAuth.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	cloneDir := filepath.Join("./backup/", project.PathWithNamespace)
	err = os.MkdirAll(cloneDir, 0755)
	if err != nil {
		return err
	}

	_, err = git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:               project.SSHURLToRepo,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              sshAuth,
	})

	return err
}

func tarGzipFolder(source, target string) error {
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	gzipWriter := gzip.NewWriter(tarfile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = filepath.Walk(source, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(file)

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		fileData, err := os.Open(file)
		if err != nil {
			return err
		}
		defer fileData.Close()

		_, err = io.Copy(tarWriter, fileData)
		return err
	})
	os.RemoveAll(source)
	return err
}

func uploadToS3(bucket, prefix, filename string) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(s3Region), // Set your AWS region
	})
	if err != nil {
		return err
	}

	uploader := s3manager.NewUploader(sess)

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(prefix + "/" + filepath.Base(filename)),
		Body:   file,
	})
	return err
}
