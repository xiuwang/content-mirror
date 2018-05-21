FROM openshift/origin-release:golang-1.9
COPY . /go/src/github.com/openshift/content-mirror/
RUN GOPATH=/go go install github.com/openshift/content-mirror/cmd/content-mirror

FROM centos:7
COPY --from=0 /go/bin/content-mirror /usr/bin/content-mirror
RUN INSTALL_PKGS=" \
      nginx \
      " && \
    yum install -y epel-release && \
    yum install -y ${INSTALL_PKGS} && rpm -V ${INSTALL_PKGS} && \
    yum clean all && \
    rm -rf /var/lib/rpm /var/lib/yum/history && \
    chmod -R uga+rwx /var/lib/nginx /var/log/nginx /run
USER 1001