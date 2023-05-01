# FROM node:20 AS build-env
FROM docker.io/node@sha256:242d81ad2a91353ac3a5ed3598582acb4a9a7761b16c60524b949a1603707848 AS build-env
ADD . /app
WORKDIR /app
RUN npm install --omit=dev

FROM distroless/nodejs20-debian11@sha256:c58f34f14c27360964cecf4cfa732dc785279e300d7fe7ae293ac633631effae

LABEL "tee.launch_policy.allow_cmd_override"="false"
LABEL "tee.launch_policy.log_redirect"="always"

COPY --from=build-env /app /app
WORKDIR /app
EXPOSE 8081
CMD ["main.js"]