FROM registry.access.redhat.com/ubi8/ubi-minimal:8.2

# Update image
RUN microdnf update -y && rm -rf /var/cache/yum

LABEL   name="quay.io/hybridappio/ham-placementrule" \
        summary="Hybrid Application Model - PlacementRule" \
        description="PlacementRule define where to deploy application components"

ENV OPERATOR=/usr/local/bin/ham-placement \
    USER_UID=1001 \
    USER_NAME=ham-placement

# install operator binary
COPY build/_output/bin/ham-placement ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# copy licenses
RUN mkdir /licenses
COPY LICENSE /licenses

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
