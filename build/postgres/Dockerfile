FROM postgres:17-alpine
COPY . .
ARG POSTGRES_DB
ARG POSTGRES_USER
ARG POSTGRES_PASSWORD
RUN sh env_var.sh DATABASE ${POSTGRES_DB} init.sql && \
    sh env_var.sh USER ${POSTGRES_USER} init.sql && \
    sh env_var.sh PASSWORD ${POSTGRES_PASSWORD} init.sql && \
    mv init.sql /docker-entrypoint-initdb.d/init.sql