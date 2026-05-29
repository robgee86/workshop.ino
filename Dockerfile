# The Go binary is statically built (CGO off) in the CI workflow and copied
# in here, so this image needs no toolchain — and `scratch` is enough.
FROM scratch
COPY workshopify /workshopify
EXPOSE 8080
ENTRYPOINT ["/workshopify"]
CMD ["-content", "/content", "-addr", ":8080"]
