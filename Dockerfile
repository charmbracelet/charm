FROM scratch
COPY charm /usr/local/bin/charm

# Create /data directory
WORKDIR /data
# Expose data volume
VOLUME /data
ENV CHARM_SERVER_DATA_DIR "/data"

# Expose ports
# SSH
EXPOSE 35353/tcp
# HTTP
EXPOSE 35354/tcp
# Stats
EXPOSE 35355/tcp
# Health
EXPOSE 35356/tcp

# Set the default command
ENTRYPOINT [ "/usr/local/bin/charm" ]
CMD [ "serve" ]