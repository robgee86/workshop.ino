FROM scratch
COPY workshop.ino /workshop.ino
EXPOSE 8080
ENTRYPOINT ["/workshop.ino"]
CMD ["-content", "/content", "-addr", ":8080"]
