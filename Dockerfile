FROM scratch
COPY workshop.ino-linux-arm64 /workshop.ino
EXPOSE 8080
ENTRYPOINT ["/workshop.ino"]
CMD ["-content", "/content", "-addr", ":8080"]
