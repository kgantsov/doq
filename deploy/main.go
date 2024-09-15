package main

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get some configuration values from the Pulumi stack, or use defaults
		cfg := config.New(ctx, "")
		k8sNamespace, err := cfg.Try("namespace")
		if err != nil {
			k8sNamespace = "default"
		}
		name, err := cfg.Try("name")
		if err != nil {
			name = "doq"
		}
		image := cfg.Require("image")
		tag := cfg.Require("tag")
		httPort := cfg.RequireInt("httPort")
		raftPort := cfg.RequireInt("raftPort")
		imagePullSecrets, err := cfg.Try("ImagePullSecrets")
		if err != nil {
			imagePullSecrets = ""
		}

		numReplicas, err := cfg.TryInt("replicas")
		if err != nil {
			numReplicas = 1
		}
		appLabels := pulumi.StringMap{
			"app": pulumi.String(name),
		}

		roleName := fmt.Sprintf("%s-endpointslice-creator", name)

		// Create a Role
		_, err = rbacv1.NewRole(ctx, roleName, &rbacv1.RoleArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: pulumi.String(k8sNamespace),
				Name:      pulumi.String(roleName),
			},
			Rules: rbacv1.PolicyRuleArray{
				&rbacv1.PolicyRuleArgs{
					ApiGroups: pulumi.StringArray{
						pulumi.String("discovery.k8s.io"),
					},
					Resources: pulumi.StringArray{
						pulumi.String("endpointslices"),
					},
					Verbs: pulumi.StringArray{
						pulumi.String("create"),
						pulumi.String("get"),
						pulumi.String("list"),
						pulumi.String("watch"),
						pulumi.String("update"),
						pulumi.String("delete"),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		corev1.NewServiceAccount(ctx, name, &corev1.ServiceAccountArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(name),
				Namespace: pulumi.String(k8sNamespace),
			},
		})

		// Create a RoleBinding
		_, err = rbacv1.NewRoleBinding(ctx, fmt.Sprintf("%s-endpointslice-creator-binding", name), &rbacv1.RoleBindingArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Namespace: pulumi.String(k8sNamespace),
				Name:      pulumi.String(fmt.Sprintf("%s-endpointslice-creator-binding", name)),
			},
			Subjects: rbacv1.SubjectArray{
				&rbacv1.SubjectArgs{
					Kind:      pulumi.String("ServiceAccount"),
					Name:      pulumi.String(name),
					Namespace: pulumi.String(k8sNamespace),
				},
			},
			RoleRef: &rbacv1.RoleRefArgs{
				Kind:     pulumi.String("Role"),
				Name:     pulumi.String(roleName),
				ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			},
		})
		if err != nil {
			return err
		}

		// Create a new Deployment with a user-specified number of replicas
		statefulSet, err := appsv1.NewStatefulSet(ctx, name, &appsv1.StatefulSetArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(name),
				Namespace: pulumi.String(k8sNamespace),
			},
			Spec: &appsv1.StatefulSetSpecArgs{
				ServiceName: pulumi.String(fmt.Sprintf("%s-internal", name)),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap(appLabels),
				},
				Replicas: pulumi.Int(numReplicas),
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap(appLabels),
					},
					Spec: &corev1.PodSpecArgs{
						ServiceAccount: pulumi.String(name),
						Containers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Image: pulumi.String(fmt.Sprintf("%s:%s", image, tag)),
								Name:  pulumi.String(name),
								Command: pulumi.StringArray{
									pulumi.String("/app"),
									pulumi.String("--cluster.service_name"),
									pulumi.String(name),
									pulumi.String("--http.port"),
									pulumi.String(fmt.Sprintf("%d", httPort)),
									pulumi.String("--raft.address"),
									pulumi.String(fmt.Sprintf("%d", raftPort)),
									pulumi.String("--storage.data_dir"),
									pulumi.String("/usr/local/doq/data"),
								},
								Ports: v1.ContainerPortArray{
									&v1.ContainerPortArgs{
										ContainerPort: pulumi.Int(httPort),
									},
								},
								VolumeMounts: corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("data"),
										MountPath: pulumi.String("/usr/local/doq/data"),
									},
								},
							},
						},
						ImagePullSecrets: corev1.LocalObjectReferenceArray{
							&corev1.LocalObjectReferenceArgs{
								Name: pulumi.String(imagePullSecrets),
							},
						},
					},
				},
				VolumeClaimTemplates: corev1.PersistentVolumeClaimTypeArray{
					&corev1.PersistentVolumeClaimTypeArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Name: pulumi.String("data"),
						},
						Spec: &corev1.PersistentVolumeClaimSpecArgs{
							AccessModes: pulumi.StringArray{
								pulumi.String("ReadWriteOnce"),
							},
							Resources: &corev1.ResourceRequirementsArgs{
								Requests: pulumi.StringMap{
									"storage": pulumi.String("5Gi"),
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// Expose the Deployment as a Kubernetes Service
		serviceInternal, err := corev1.NewService(ctx, fmt.Sprintf("%s-internal", name), &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(fmt.Sprintf("%s-internal", name)),
				Namespace: pulumi.String(k8sNamespace),
				// Labels:    pulumi.StringMap(appLabels),
			},
			Spec: &corev1.ServiceSpecArgs{
				ClusterIP:                pulumi.String("None"),
				PublishNotReadyAddresses: pulumi.Bool(true),
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(httPort),
						TargetPort: pulumi.Any(httPort),
						Protocol:   pulumi.String("TCP"),
					},
				},
				Selector: pulumi.StringMap(appLabels),
			},
		})
		if err != nil {
			return err
		}

		// Expose the Deployment as a Kubernetes Service
		service, err := corev1.NewService(ctx, name, &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(name),
				Namespace: pulumi.String(k8sNamespace),
				Labels:    pulumi.StringMap(appLabels),
			},
			Spec: &corev1.ServiceSpecArgs{
				ClusterIP: pulumi.String("None"),
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(httPort),
						TargetPort: pulumi.Any(httPort),
						Protocol:   pulumi.String("TCP"),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// Export some values for use elsewhere
		ctx.Export("statefulSettName", statefulSet.Metadata.Name())
		ctx.Export("serviceName", serviceInternal.Metadata.Name())
		ctx.Export("serviceName", service.Metadata.Name())

		return nil
	})
}
