FROM 416670754337.dkr.ecr.eu-west-2.amazonaws.com/ci-golang-build-1.24:latest

WORKDIR /opt

RUN mkdir -p assets

RUN ls -R /

COPY /app .
COPY product_code.yml ./assets/
COPY docker_start.sh .

RUN ls -R /opt

RUN chmod +x docker_start.sh
CMD ["./docker_start.sh"]
