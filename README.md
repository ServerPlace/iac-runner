# iac-runner

Este repositório contém as definições para a construção e publicação da imagem Docker do `iac-runner`, além das
diretrizes de configuração necessárias para sua execução em clusters Kubernetes.

## Visão Geral

O `iac-runner` atua como o componente de execução de tarefas em um fluxo de infraestrutura como código (IaC). Para seu
funcionamento pleno, é indispensável a integração com o componente controlador do ecossistema chamado `iac-controller`
que é pré-requisito a este projeto.

## Pré-requisitos

Antes de iniciar a configuração dos recursos deste projeto, certifique-se de atender aos seguintes requisitos:

1.  **Dependência de Projeto:** O projeto `iac-controller` deve estar devidamente configurado e publicado, uma vez que o
    `iac-runner` depende de configurações básicas estabelecidas por ele.
2.  **Registry de Imagens:** É necessário possuir um repositório Docker privado para as imagens do `iac-runner`.
3.  **Cluster Kubernetes:** Assumimos que existe um cluster pronto para uso.
3.  **Permissões de Cluster:** Lembre-se, o cluster Kubernetes deve possuir permissões de acesso à internet seu
    funcionamento, além de permissão de leitura no repositório docker criado acima (`imagePullSecrets` ou permissões de
    IAM) para realizar o download das imagens.

## Build e Publicação no Registry Privado

O processo de compilação do código fonte em Go é realizado de forma multi-stage diretamente durante o build da imagem Docker.
É necessário que Go e Docker estejam instalados para seu funcionamento.

> **NOTA:** Talvez seja necessário comandos de login no registry antes de realizar o docker push abaixo. Certifique na
> documentação do seu registry antes de executar os passos abaixo.

Utilize o exemplo em [./scripts/01-build-and-push.sh](./scripts/01-build-and-push.sh) para build e push da imagem docker ao repositório.

## Configuração de Infraestrutura e Permissionamentos

### Permissionamentos na Cloud

1. Em ambientes Cloud, é necessário realizar configurações para permitir que o cluster Kubernetes tenha acesso de
   leitura ao Registry no qual a imagem docker do `iac-runner` foi armazenada. Verifique permissionamento das Services
   Accounts dos nós e serviços Kubernetes na Cloud.

1. Também é necessário que o agente `iac-runner` tenha permissão de acesso à Internet para acesso aos repositórios IACs
   que ele irá operar.

### Permissionamentos no Kubernetes

Como o `iac-controller` será executado dentro de um GCP Cloud Run, também é necessário que o `iac-runner` tenha
permissões para acesso ao serviço de Workload Identity no Google Cloud Platform (GCP). É necessário vincular uma
Google Service Account (GSA) à Kubernetes Service Account (KSA) utilizada pelo `iac-runner`.

Certifique que o **cluster** possui um Kubernetes Service Account configurado para o `iac-runner` com a `annotation`
configurada corretamente. Em  [./scripts/02-annotate-ksa.sh](./scripts/02-annotate-ksa.sh) possui-se um comando de exemplo para associar uma KSA, via
`annotation` a um GSA no GCP.

Já em [./scripts/03-create-gsa-with-workloadidentity.sh](./scripts/03-create-gsa-with-workloadidentity.sh), exemplifica a atribuição da role
`roles/iam.workloadIdentityUser` para permitir que a KSA assuma a identidade da GSA dentro do **GCP**.

Informações complementares:

- `<KSA_NAME>`: Nome do Kubernetes Service Account dentro do namespace `<K8S_NAMESPACE>` no qual o `iac-runner` está
  associado.
- `<K8S_NAMESPACE>`: Nome do namespace, dentro do Kubernetes, no qual os pods do `iac-runner` estão sendo executados.
- `<PROJECT_ID>`: Nome do Projeto no GCP na qual o cluster Kubernetes se encontra.
- `<GSA_NAME>`: Nome da Google Service Account criada para que o `iac-runner` utilize recursos no GCP.

Faça a alteração acima seguindo orientações da sua respectiva Cloud/infraestrutura para concluir o objetivo de
comunicação entre o `iac-runner` e o `iac-controller`.


## Deploy do `iac-runner` no Kubernetes

Neste momento é configurado, dentro do cluster, o deploy do `iac-runner`. Neste exemplo utilizaremos a publicação do
agente `iac-runner` utilizando o KEDA como escalonamento dinâmicos dos jobs dos agentes.

> As principais informações para o funcionamento do agente são definidas via variáveis de ambiente (`env`) na publicação
  do workload no Kubernetes e serão demonstradas abaixo.

### Autoscaling com KEDA

Será configurado um [KEDA](https://keda.sh/), no Kubernetes, que fará o escalonamento de pods do agent com base na fila
de processos da pipeline a serem executados pelo [Azure
Devops](https://techcommunity.microsoft.com/blog/desenvolvedoresbr/utilizando-keda-para-escalar-agents-no-azure-devops-no-kubernetes/3717045).

Para que o KEDA funcione configure no cluster os itens abaixo:

1. `Kubernetes Secret Object` contendo informações para que o Keda acesse o Azure Pipeline e colha informações para
   escalonamento de runners. Código de exemplo disponível em [./scripts/04-secret.yaml](./scripts/04-secret.yaml).

1. `Keda TriggerAuthentication` na qual utilizará a secret acima para acesso, exemplo disponível em
   [./scripts/05-trigger-authentication.yaml](./scripts/05-trigger-authentication.yaml).

1. `Keda ScaledJob` que terá todas as informações e configurações para que o agente `iac-runner` possa conectar e
   executar suas tarefas. Disponível em [./scripts/06-scaled-job.yaml](./scripts/06-scaled-job.yaml).
