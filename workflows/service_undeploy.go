package workflows

import (
	"fmt"
	"github.com/stelligent/mu/common"
	"strings"
)

// NewServiceUndeployer create a new workflow for undeploying a service in an environment
func NewServiceUndeployer(ctx *common.Context, serviceName string, environmentName string) Executor {

	workflow := new(serviceWorkflow)

	return newPipelineExecutor(
		workflow.serviceInput(ctx, serviceName),
		workflow.serviceUndeployer(ctx.Config.Namespace, environmentName, ctx.StackManager, ctx.StackManager),
	)
}

func (workflow *serviceWorkflow) serviceUndeployer(namespace string, environmentName string, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter) Executor {
	return func() error {
		log.Noticef("Undeploying service '%s' from '%s'", workflow.serviceName, environmentName)
		svcStackName := common.CreateStackName(namespace, common.StackTypeService, workflow.serviceName, environmentName)
		svcStack := stackWaiter.AwaitFinalStatus(svcStackName)
		if svcStack != nil {
			err := stackDeleter.DeleteStack(svcStackName)
			if err != nil {
				return err
			}
			svcStack = stackWaiter.AwaitFinalStatus(svcStackName)
			if svcStack != nil && !strings.HasSuffix(svcStack.Status, "_COMPLETE") {
				return fmt.Errorf("Ended in failed status %s %s", svcStack.Status, svcStack.StatusReason)
			}
		} else {
			log.Info("  Stack is already deleted.")
		}

		return nil
	}
}

func (workflow *serviceWorkflow) serviceRolesetTerminator(rolesetDeleter common.RolesetDeleter, environmentName string) Executor {
	return func() error {
		err := rolesetDeleter.DeleteServiceRoleset(environmentName, workflow.serviceName)
		if err != nil {
			return err
		}

		return nil
	}
}
