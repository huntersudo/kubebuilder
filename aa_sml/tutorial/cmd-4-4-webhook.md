

### 4.4 webhook

https://cloudnative.to/kubebuilder/reference/webhook-overview.html

Webhooks 是一种以阻塞方式发送的信息请求。实现 webhooks 的 web 应用程序将在特定事件发生时向其他应用程序发送 HTTP 请求。

在 kubernetes 中，有下面三种 webhook：
- admission webhook， https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks
- authorization webhook ，https://kubernetes.io/docs/reference/access-authn-authz/webhook/
- CRD conversion webhook。https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#webhook-conversion

在 controller-runtime 库中，我们支持 admission webhooks 和 CRD conversion webhooks。
https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/webhook


Kubernetes 在 1.9 版本中（该特性进入 beta 版时）支持这些动态 admission webhooks。
Kubernetes 在 1.15 版本（该特性进入 beta 版时）支持 conversion webhook。

### 4.4.1 准入 Webhooks
https://cloudnative.to/kubebuilder/reference/admission-webhook.html

准入 webhook 是 HTTP 的回调，它可以接受准入请求，处理它们并且返回准入响应。

Kubernetes 提供了下面几种类型的准入 webhook：
- 变更准入 Webhook, 这种类型的 webhook 会在对象创建或是更新且没有存储前改变操作对象，然后才存储。
  它可以用于资源请求中的默认字段，比如在 Deployment 中没有被用户制定的字段。它可以用于注入 sidecar 容器。
- 验证准入 Webhook, 这种类型的 webhook 会在对象创建或是更新且没有存储前验证操作对象，然后才存储。
  它可以有比纯基于 schema 验证更加复杂的验证。比如：交叉字段验证和 pod 镜像白名单。

默认情况下 apiserver 自己没有对 webhook 进行认证。然而，如果你想认证客户端，你可以配置 apiserver 使用基本授权，持有 token，或者证书对 webhook 进行认证。 详细的步骤可以查看这里。
https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers

### 4.4.2 核心类型的准入 Webhook

为 CRD 构建准入 webhook 非常容易，这在 CronJob 教程中已经介绍过了。

由于 kubebuilder 不支持核心类型的 webhook 自动生成，您必须使用 controller-runtime 的库来处理它。这里可以参考 controller-runtime 的一个 示例。
https://github.com/kubernetes-sigs/controller-runtime/tree/master/examples/builtins


建议使用 kubebuilder 初始化一个项目，然后按照下面的步骤为核心类型添加准入 webhook。

#### 实现处理程序
你需要用自己的处理程序去实现 admission.Handler 接口。

``` 
type podAnnotator struct {
    Client  client.Client
    decoder *admission.Decoder
}

func (a *podAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
    pod := &corev1.Pod{}
    err := a.decoder.Decode(req, pod)
    if err != nil {
        return admission.Errored(http.StatusBadRequest, err)
    }

    //在 pod 中修改字段

    marshaledPod, err := json.Marshal(pod)
    if err != nil {
        return admission.Errored(http.StatusInternalServerError, err)
    }
    return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
```
如果需要客户端，只需在结构构建时传入客户端。
如果你为你的处理程序添加了 InjectDecoder 方法，将会注入一个解码器。
``` 
func (a *podAnnotator) InjectDecoder(d *admission.Decoder) error {
    a.decoder = d
    return nil
}
```
注意: 为了使得 controller-gen 能够为你生成 webhook 配置，你需要添加一些标记。
例如， // +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io


#### 更新 main.go
现在你需要在 webhook 服务端中注册你的处理程序。

``` 
mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{Handler: &podAnnotator{Client: mgr.GetClient()}})

```
您需要确保这里的路径与标记中的路径相匹配。

#### 部署
部署它就像为 CRD 部署 webhook 服务端一样。你需要
- 提供服务证书
- 部署服务端






















































