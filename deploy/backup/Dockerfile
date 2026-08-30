# syntax=docker/dockerfile:1
#
# pg_dump must be >= the server it dumps, so this image is pinned to the SAME
# major as the postgres service. Bump both together — a version skew here fails
# at the moment you can least afford it.
FROM postgres:17-alpine

# rclone is one static binary that speaks S3, Cloudflare R2, Backblaze B2 and
# DigitalOcean Spaces identically, so the bucket choice stays a config decision
# rather than something baked into this image.
RUN apk add --no-cache rclone tzdata

COPY backup.sh  /usr/local/bin/backup.sh
COPY restore.sh /usr/local/bin/restore.sh
COPY crontab    /etc/crontabs/root
RUN chmod +x /usr/local/bin/backup.sh /usr/local/bin/restore.sh

# -f foreground, -d 8 logs to stderr so `docker logs` shows runs and failures.
# Without -d, crond logs to a syslog nothing is reading and a failing backup is
# silent — which is the same as having no backup.
CMD ["crond", "-f", "-d", "8"]
