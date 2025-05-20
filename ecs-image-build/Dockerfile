FROM 416670754337.dkr.ecr.eu-west-2.amazonaws.com/ci-golang-build-1.24:latest

WORKDIR /opt

RUN mkdir -p assets

COPY app/payment-reconciliation-consumer .
COPY app/docker_start.sh .
COPY app/product_code.yml ./assets/

RUN chmod +x docker_start.sh
CMD ["./docker_start.sh"]