{{- define "era-one.golangService" -}}
{{- $root := index . 0 -}}
{{- $name := index . 1 -}}
{{- $svc := index . 2 -}}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "era-one.fullname" $root }}-{{ $name }}
spec:
  ports:
    - port: {{ $svc.port }}
      targetPort: http
      name: http
    {{- if $svc.grpcPort }}
    - port: {{ $svc.grpcPort }}
      targetPort: grpc
      name: grpc
    {{- end }}
  selector:
    app.kubernetes.io/name: {{ include "era-one.name" $root }}
    app.kubernetes.io/component: {{ $name }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "era-one.fullname" $root }}-{{ $name }}
spec:
  replicas: {{ default 1 $svc.replicas }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ include "era-one.name" $root }}
      app.kubernetes.io/component: {{ $name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ include "era-one.name" $root }}
        app.kubernetes.io/component: {{ $name }}
    spec:
      containers:
        - name: {{ $name }}
          image: {{ $svc.image }}
          imagePullPolicy: {{ $root.Values.global.imagePullPolicy }}
          ports:
            - containerPort: {{ $svc.port }}
              name: http
            {{- if $svc.grpcPort }}
            - containerPort: {{ $svc.grpcPort }}
              name: grpc
            {{- end }}
          env:
            - name: ERA_HTTP_ADDR
              value: ":{{ $svc.port }}"
            {{- range $k, $v := $svc.env }}
            - name: {{ $k }}
              value: {{ $v | quote }}
            {{- end }}
          livenessProbe:
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 10
            periodSeconds: 20
          readinessProbe:
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
          {{- if $svc.persistence }}
          volumeMounts:
            - name: data
              mountPath: {{ default "/data" $svc.persistence.mountPath }}
          {{- end }}
      {{- if $svc.persistence }}
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: {{ include "era-one.fullname" $root }}-{{ $name }}
      {{- end }}
{{- if $svc.persistence }}
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ include "era-one.fullname" $root }}-{{ $name }}
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: {{ default "5Gi" $svc.persistence.size }}
{{- end }}
{{- end }}
