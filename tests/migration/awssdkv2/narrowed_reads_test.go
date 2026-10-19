/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2_test

import (
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// in4it/ecs-deploy's shape, typed against v1: an ECS Task's Group, which is no
// enum though EC2's permissions' Group is; a container's exit code, which v2
// narrowed and still points to; and a revision v2 narrowed and holds by value.
// Each is read at v2's width and widened back to v1's.
func TestMigrateNarrowedReadsTyped(t *testing.T) {
	requireModuleCache(t, "github.com/aws/aws-sdk-go@v1.55.8")
	migrateSpec().RewriteRun(t,
		test.GoProject("grp",
			test.GoMod(`
			module example.com/grp

			go 1.24

			require github.com/aws/aws-sdk-go v1.55.8
		`, `
			module example.com/grp

			go 1.24

			require (
				github.com/aws/aws-sdk-go-v2 v1.47.3
				github.com/aws/aws-sdk-go-v2/service/ecs v1.100.3
			)

			require (
				github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 // indirect
				github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 // indirect
				github.com/aws/smithy-go v1.28.5 // indirect
			)
		`),
			test.Golang(`
			package ecs

			import (
				"time"

				"github.com/aws/aws-sdk-go/aws"
				"github.com/aws/aws-sdk-go/service/ecs"
			)

			type RunningTask struct {
				Group      string
				ExitCode   int64
				StoppedAt  time.Time
			}

			func tasks(svc *ecs.ECS, cluster string) ([]RunningTask, error) {
				var out []RunningTask
				result, err := svc.DescribeTasks(&ecs.DescribeTasksInput{Cluster: aws.String(cluster)})
				if err != nil {
					return nil, err
				}
				for _, task := range result.Tasks {
					rs := RunningTask{}
					if task.Group != nil {
						rs.Group = *task.Group
					}
					for _, container := range task.Containers {
						if container.ExitCode != nil {
							rs.ExitCode = *container.ExitCode
						}
					}
					out = append(out, rs)
				}
				return out, nil
			}

			func revision(svc *ecs.ECS, family string) (int64, error) {
				result, err := svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{TaskDefinition: aws.String(family)})
				if err != nil {
					return 0, err
				}
				return aws.Int64Value(result.TaskDefinition.Revision), nil
			}
		`, `
			package ecs

			import (
				"context"
				"time"

				"github.com/aws/aws-sdk-go-v2/aws"
				"github.com/aws/aws-sdk-go-v2/service/ecs"
			)

			type RunningTask struct {
				Group      string
				ExitCode   int64
				StoppedAt  time.Time
			}

			func tasks(svc *ecs.Client, cluster string) ([]RunningTask, error) {
				var out []RunningTask
				result, err := svc.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{Cluster: aws.String(cluster)})
				if err != nil {
					return nil, err
				}
				for _, task := range awsCompatPtrs(result.Tasks) {
					rs := RunningTask{}
					if task.Group != nil {
						rs.Group = *task.Group
					}
					for _, container := range awsCompatPtrs(task.Containers) {
						if container.ExitCode != nil {
							rs.ExitCode = int64(*container.ExitCode)
						}
					}
					out = append(out, rs)
				}
				return out, nil
			}

			func revision(svc *ecs.Client, family string) (int64, error) {
				result, err := svc.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{TaskDefinition: aws.String(family)})
				if err != nil {
					return 0, err
				}
				return int64(result.TaskDefinition.Revision), nil
			}
		`),
			test.Generated("zz_awssdkv2_compat.go", compatHelpersIn("ecs")),
		),
	)
}
