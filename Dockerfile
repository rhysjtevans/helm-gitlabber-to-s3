FROM python:3-alpine
WORKDIR /app

RUN apk add --no-cache bash git aws-cli openssh-client && pip3 install gitlabber

COPY init.sh .
RUN chmod +x ./init.sh
ENTRYPOINT /bin/bash -c /app/init.sh
# CMD /app/init.sh