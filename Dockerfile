FROM quay.io/app-sre/golang:1.17.0 as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN make gobuild

# Issue with gpg --import when using this image
# Error related to pinentry missing
# Using golang image for development and will return to resolve this before production

#FROM quay.io/app-sre/ubi8-ubi-minimal:8.6
#COPY --from=builder /app/git-sync-pull /
#COPY run.sh .
#CMD ["bash", "run.sh"]

ENTRYPOINT [ "./git-sync-pull" ]