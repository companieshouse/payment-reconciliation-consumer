FROM 416670754337.dkr.ecr.eu-west-2.amazonaws.com/ci-golang-build-1.24:latest

WORKDIR /opt

RUN mkdir -p assets

COPY /app .
RUN ls -al 
COPY docker_start.sh .
COPY product_code.yml ./assets/


RUN chmod +x docker_start.sh
CMD ["./docker_start.sh"]bu