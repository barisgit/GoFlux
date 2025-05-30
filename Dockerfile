FROM scratch

# Copy the binary from the build context
COPY flux /usr/local/bin/flux

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/flux"]

# Default command
CMD ["--help"] 