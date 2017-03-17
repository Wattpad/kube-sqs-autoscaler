FROM alpine:3.4

RUN  apk add --no-cache --update ca-certificates

COPY kube-sqs-autoscaler /

CMD ["/kube-sqs-autoscaler"]
