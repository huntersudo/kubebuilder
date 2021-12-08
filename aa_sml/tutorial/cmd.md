

https://book.kubebuilder.io/quick-start.html

https://cloudnative.to/kubebuilder/quick-start.html



###创建项目
正如 快速入门 中所介绍的那样，我们需要创建一个新的项目。确保你已经 安装 Kubebuilder，然后再创建一个新项目。
``` 
# 我们将使用 tutorial.kubebuilder.io 域，
# 所以所有的 API 组将是<group>.tutorial.kubebuilder.io.
kubebuilder init --domain tutorial.kubebuilder.io
```

现在我们已经创建了一个项目，让我们来看看 Kubebuilder 为我们初始化了哪些组件。....

### 一个 kubebuilder 项目有哪些组件？
当自动生成一个新项目时，Kubebuilder 为我们提供了一些基本的模板。

创建基础组件
首先是基本的项目文件初始化，为项目构建做好准备。

`go.mod`: 我们的项目的 Go mod 配置文件，记录依赖库信息。
```
go 1.15

require (
    github.com/go-logr/logr v0.1.0
    github.com/onsi/ginkgo v1.12.1
    github.com/onsi/gomega v1.10.1
    github.com/robfig/cron v1.2.0
    k8s.io/api v0.18.6
    k8s.io/apimachinery v0.18.6
    k8s.io/client-go v0.18.6
    sigs.k8s.io/controller-runtime v0.6.2
) 
```
`Makefile`: 用于控制器构建和部署的 Makefile 文件
`PROJECT`: 用于生成组件的 Kubebuilder 元数据
``` 
domain: tutorial.kubebuilder.io
layout: go.kubebuilder.io/v3-alpha
projectName: project
repo: tutorial.kubebuilder.io/project
resources:
- group: batch
  kind: CronJob
  version: v1
version: 3-alpha
```

#### 启动配置

我们还可以在 config/ 目录下获得启动配置。现在，它只包含了在集群上启动控制器所需的 Kustomize YAML 定义，但一旦我们开始编写控制器，
它还将包含我们的 CustomResourceDefinitions(CRD) 、RBAC 配置和 WebhookConfigurations 。

config/default 在标准配置中包含 Kustomize base ，它用于启动控制器。

其他每个目录都包含一个不同的配置，重构为自己的基础。
- config/manager: 在集群中以 pod 的形式启动控制器
- config/rbac: 在自己的账户下运行控制器所需的权限

#### 入口函数

最后，当然也是最重要的一点，生成项目的入口函数：main.go。接下来我们看看它。.....

### kubebuilder-3.2.0 +go16 版本

````  
go mod init tutorial
kubebuilder init --domain tutorial.kubebuilder.io
kubebuilder create api --group batch --version v1 --kind CronJob

````

#### webhook

如果你想为你的 CRD 实现一个 admission webhooks， 你需要做的一件事就是去实现Defaulter 和/或 Validator 接口。
Kubebuilder 会帮你处理剩下的事情，像下面这些：
创建 webhook 服务端。
确保服务端已添加到 manager 中。
为你的 webhooks 创建处理函数。
用路径在你的服务端中注册每个处理函数。

````
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation

````

#### 部署 cert manager
我们建议使用 cert manager 为 webhook 服务器提供证书。只要其他解决方案将证书放在期望的位置，也将会起作用。
你可以按照 cert manager 文档 进行安装。
Cert manager 还有一个叫做 CA 注入器的组件，该组件负责将 CA 捆绑注入到 Mutating|ValidatingWebhookConfiguration 中。

为此，你需要在 Mutating|ValidatingWebhookConfiguration 对象中使用带有 key 为 cert-manager.io/inject-ca-from 的注释。 注释的值应指向现有的证书 CR 实例，
格式为 <certificate-namespace>/<certificate-name>。
这是我们用于注释 Mutating|ValidatingWebhookConfiguration 对象的 kustomize patch。

``` 
# This patch add annotation to admission webhook config and
# the variables $(CERTIFICATE_NAMESPACE) and $(CERTIFICATE_NAME) will be substituted by kustomize.

apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: mutating-webhook-configuration
  annotations:
    cert-manager.io/inject-ca-from: $(CERTIFICATE_NAMESPACE)/$(CERTIFICATE_NAME)
---
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: validating-webhook-configuration
  annotations:
    cert-manager.io/inject-ca-from: $(CERTIFICATE_NAMESPACE)/$(CERTIFICATE_NAME)
    
```
























