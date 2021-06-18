FROM gcr.io/distroless/static
COPY . /
CMD ["/notification_relay"]