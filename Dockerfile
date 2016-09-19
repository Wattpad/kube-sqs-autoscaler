FROM alpine:3.4

COPY kube-sqs-autoscaler /

CMD ["/kube-sqs-autoscaler"]
