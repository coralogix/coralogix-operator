FROM registry.access.redhat.com/ubi8/ubi:latest

ARG VERSION=1.0.0
ARG RELEASE=1

LABEL name="coralogix-fluentd" \
      vendor="Coralogix Inc." \
      version="${VERSION}" \
      release="${RELEASE}" \
      summary="Special FluentD image for Coralogix integration" \
      description="Special FluentD image for Coralogix integration"

RUN yum install -y hostname sudo make gcc gcc-c++ ruby-devel && \
    curl -L https://toolbelt.treasuredata.com/sh/install-redhat-td-agent4.sh | sh && \
    td-agent-gem install \
        fluent-plugin-kubernetes_metadata_filter \
        fluent-plugin-multi-format-parser \
        fluent-plugin-detect-exceptions && \
    td-agent-gem sources --clear-all && \
    rm -rf /tmp/* /var/tmp/* /var/cache/* /opt/td-agent/embedded/lib/ruby/gems/*/cache/*.gem && \
    ulimit -n 65536 && \
    yum autoremove -y make gcc gcc-c++ ruby-devel && \
    yum clean all

ENV LD_PRELOAD=/opt/td-agent/embedded/lib/libjemalloc.so

USER td-agent

COPY licenses /licenses
COPY configs/td-agent.conf /etc/td-agent/td-agent.conf

CMD ["td-agent"]
