# spot-handler-service
Spot interruption notice can be used to drain a node 2 min earlier then the actual interuption. This will allow gracefull handling of the services running on the node. 

The can we extended to other tasks like draing from elb and gracefull shutdown to servers aswell.  

For now we will be using kubectl to drain node

Steps:
- Register cloudwatch event for spot interruption
- Mark target as sns for notification, Add notice url to sns and check for incomming request log. use confirmation link from message body gor confirmation
- Subscribe http call to service. Can depend on sqs as this might delay the request. 
- Use notice api to confirm request and process notification
- To enable kubectl on pod, allow access to kube config from node and install kubectl on pod   
- Shedule pod on ondemand machine


# Sample deployment yaml
```
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  annotations:
  labels:
    app: spot-handler-service
    group: infra
    owner: shubham
    tier: backend
  name: spot-handler-service
  namespace: production
spec:
  minReadySeconds: 10
  progressDeadlineSeconds: 600
  replicas: 2
  revisionHistoryLimit: 5
  selector:
    matchLabels:
      app: spot-handler-service
      group: infra
      owner: shubham
      tier: backend
  strategy:
    rollingUpdate:
      maxSurge: 0
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      annotations:
        prometheus.io/path: /
        prometheus.io/port: "8080"
        prometheus.io/scrape: "false"
        slack: '@shubham negi'
      creationTimestamp: null
      labels:
        app: spot-handler-service
        group: infra
        owner: shubham
        tier: backend
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kops.k8s.io/instance-life-cycle
                operator: NotIn
                values:
                - spot
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - spot-handler-service
              topologyKey: kubernetes.io/hostname
            weight: 100
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - spot-handler-service
            topologyKey: kubernetes.io/hostname
      containers:
      - env:
        - name: ENDPOINTS_HEALTH_ENABLED
          value: "true"
        - name: ENDPOINTS_AUTOCONFIG_ENABLED
          value: "false"
        - name: ENDPOINTS_BEANS_ENABLED
          value: "false"
        - name: ENDPOINTS_CONFIGPROPS_ENABLED
          value: "false"
        - name: ENDPOINTS_DUMP_ENABLED
          value: "false"
        - name: ENDPOINTS_ENV_ENABLED
          value: "false"
        - name: ENDPOINTS_INFO_ENABLED
          value: "false"
        - name: ENDPOINTS_METRICS_ENABLED
          value: "false"
        - name: ENDPOINTS_MAPPINGS_ENABLED
          value: "false"
        - name: ENDPOINTS_SHUTDOWN_ENABLED
          value: "false"
        - name: ENDPOINTS_TRACE_ENABLED
          value: "false"
        - name: ENDPOINTS_HEAPDUMP_ENABLED
          value: "false"
        - name: ENDPOINTS_ERROR_ENABLED
          value: "false"
        - name: MANAGEMENT_HEALTH_DEFAULTS_ENABLED
          value: "false"        
        - name: K8S_SERVICE_NAME
          value: spot-handler-service
        - name: GET_HOSTS_FROM
          value: dns        
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              key: key
              name: limetray-aws-readonly
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              key: secret
              name: limetray-aws-readonly
        - name: AWS_REGION
          valueFrom:
            secretKeyRef:
              key: region
              name: limetray-aws-readonly
        - name: KUBECTL_CONFIG
          value: /var/lib/kubelet/kubeconfig
        - name: GIN_MODE
          value: release
        - name: SLACK_URL
          valueFrom:
            secretKeyRef:
              key: url
              name: limetray-slack
        image: 445897275450.dkr.ecr.ap-southeast-1.amazonaws.com/spot-handler-service:master.1.0.0.b20-1637744094934
        imagePullPolicy: IfNotPresent
        livenessProbe:
          failureThreshold: 30
          httpGet:
            path: /health
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 90
          periodSeconds: 20
          successThreshold: 1
          timeoutSeconds: 10
        name: spot-handler-service
        ports:
        - containerPort: 8080
          protocol: TCP
        readinessProbe:
          failureThreshold: 30
          httpGet:
            path: /health
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 30
          periodSeconds: 20
          successThreshold: 1
          timeoutSeconds: 10
        resources:
          requests:
            cpu: 10m
            memory: 50Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        volumeMounts:
        - mountPath: /var/lib/kubelet/kubeconfig
          name: kube-volume
      dnsConfig:
        options:
        - name: ndots
          value: "1"
        - name: single-request-reopen
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 60
      tolerations:
      - effect: NoSchedule
        key: dedicated
        operator: Equal
        value: nodes-compute
      - effect: NoSchedule
        key: dedicated
        operator: Equal
        value: cpos-nodes
      - effect: NoSchedule
        key: dedicated
        operator: Equal
        value: platform-nodes
      volumes:
      - hostPath:
          path: /var/lib/kubelet/kubeconfig
          type: File
        name: kube-volume
```
