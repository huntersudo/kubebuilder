
### Kubebuilder 项目结构之间的迁移通常会涉及到一些手动操作。

https://cloudnative.to/kubebuilder/migrations.html

#### Kubebuilder v1 版本 VS v2 版本
这篇文档涵盖了从 v1 版本迁移到 v2 版本时所有破坏性的变化。

所有细节变化（破坏性的或者其他）可以查询 controller-runtime, controller-tools 和 kubebuilder 发布说明。

常规
V2 版本项目中使用 go modules。但是 kubebuilder 会继续支持 dep 直到 go 1.13 正式发布。

controller-runtime
- Client.List 现在使用 functional options (List(ctx, list, ...option)) 代替 List(ctx, ListOptions, list)。
- Client 接口加入了 Client.DeleteAllOf。
- 默认开启 Metrics。
- pkg/runtime下的一些包已经被移除，它们旧的位置已被弃用。并将会在 controller-runtime v1.0.0 版本之前删除。更多信息请看 godocs。

Webhook-related
- webhooks 的自动证书生成已经被移除，并且它将不再自动注册。使用 controller-tools 去生成 webhook 配置。如果你需要生成证书，我们推荐使用 cert-manager。
 Kubebuilder v2 版本将会自动生成证书管理器配置供你使用 -- 更多细节请看 Webhook 教程。
- builder 包现在为 controllers 和 webhooks 提供了独立的生成器，这便于选择哪个去运行。

controller-tools
- 在 v2 版本中已经重写了生成器框架。在许多情况下，它仍然像以前一样工作，但是要注意有一些破坏性的变化。更多细节请看 marker 文档。
  
Kubebuilder
- Kubebuilder v2 版本引入了简化的项目布局。你可以在 这里 找到相关设计文档。
  https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/simplified-scaffolding.md

- 在 v1 版本中，manager 作为一个 StatefulSet 部署，而在 v2 版本中是作为一个 Deployment 部署。
- kubebuilder create webhook 命令被用来自动生成 mutating/validating/conversion webhooks. 它代替了 kubebuilder alpha webhook 命令。
- v2 版本使用 distroless/static 代替 Ubuntu 作为基础镜像。这减少了镜像大小和受攻击面。
- v2 版本要求 kustomize v3.1.0+。


#### v1 迁移v2 指南
https://cloudnative.to/kubebuilder/migration/guide.html


```
Webhook 标记
// 这些是 v2 标记

// 这些是关于 mutating webhook 的
// +kubebuilder:webhook:path=/mutate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=true,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=mcronjob.kb.io

...

// 这些是关于 validating webhook 的
// +kubebuilder:webhook:path=/validate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=false,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=vcronjob.kb.io
```

默认的 verbs 是 verbs=create;update。我们需要确保 verbs 和我们所需要的是一致的。 比如，如果我们仅仅想验证 creation，那么我们就需要将 verbs 改成 verbs=create。
在 v1 中，一个单个的 webhook 标记可能会被拆分成多个段落。但是在 v2 中，每一个 webhook 必须由一个单个的标记来表示。


