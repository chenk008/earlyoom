FROM golang as build

WORKDIR /usr/src
COPY . .

ENV CFLAGS -static
RUN make

###

FROM ubuntu
COPY --from=build /usr/src/earlyoom /

COPY --from=build /usr/src/k8s_event /k8s_event

ENTRYPOINT ["/earlyoom"]
