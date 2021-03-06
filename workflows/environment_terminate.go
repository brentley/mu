package workflows

import (
	"fmt"
	"github.com/stelligent/mu/common"
	"strings"
)

// NewEnvironmentTerminator create a new workflow for terminating an environment
func NewEnvironmentTerminator(ctx *common.Context, environmentName string) Executor {

	workflow := new(environmentWorkflow)

	return newPipelineExecutor(
		workflow.environmentServiceTerminator(environmentName, ctx.StackManager, ctx.StackManager, ctx.StackManager, ctx.RolesetManager),
		workflow.environmentDbTerminator(environmentName, ctx.StackManager, ctx.StackManager, ctx.StackManager),
		workflow.environmentEcsTerminator(ctx.Config.Namespace, environmentName, ctx.StackManager, ctx.StackManager),
		workflow.environmentConsulTerminator(ctx.Config.Namespace, environmentName, ctx.StackManager, ctx.StackManager),
		workflow.environmentRolesetTerminator(ctx.RolesetManager, environmentName),
		workflow.environmentElbTerminator(ctx.Config.Namespace, environmentName, ctx.StackManager, ctx.StackManager),
		workflow.environmentVpcTerminator(ctx.Config.Namespace, environmentName, ctx.StackManager, ctx.StackManager),
	)
}

func (workflow *environmentWorkflow) environmentServiceTerminator(environmentName string, stackLister common.StackLister, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter, rolesetDeleter common.RolesetDeleter) Executor {
	return func() error {
		log.Noticef("Terminating Services for environment '%s' ...", environmentName)
		stacks, err := stackLister.ListStacks(common.StackTypeService)
		if err != nil {
			return err
		}
		for _, stack := range stacks {
			if stack.Tags["environment"] != environmentName {
				continue
			}
			err := stackDeleter.DeleteStack(stack.Name)
			if err != nil {
				return err
			}

			rolesetDeleter.DeleteServiceRoleset(environmentName, stack.Tags["service"])
		}
		for _, stack := range stacks {
			if stack.Tags["environment"] != environmentName {
				continue
			}
			log.Infof("   Undeploying service '%s' from environment '%s'", stack.Tags["service"], environmentName)
			stackWaiter.AwaitFinalStatus(stack.Name)
		}

		return nil
	}
}
func (workflow *environmentWorkflow) environmentDbTerminator(environmentName string, stackLister common.StackLister, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter) Executor {
	return func() error {
		log.Noticef("Terminating Databases for environment '%s' ...", environmentName)
		stacks, err := stackLister.ListStacks(common.StackTypeDatabase)
		if err != nil {
			return err
		}
		for _, stack := range stacks {
			if stack.Tags["environment"] != environmentName {
				continue
			}
			err := stackDeleter.DeleteStack(stack.Name)
			if err != nil {
				return err
			}
		}
		for _, stack := range stacks {
			if stack.Tags["environment"] != environmentName {
				continue
			}
			log.Infof("   Terminating database for service '%s' from environment '%s'", stack.Tags["service"], environmentName)
			stackWaiter.AwaitFinalStatus(stack.Name)
		}

		return nil
	}
}
func (workflow *environmentWorkflow) environmentConsulTerminator(namespace string, environmentName string, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter) Executor {
	return func() error {
		log.Noticef("Terminating Consul environment '%s' ...", environmentName)
		envStackName := common.CreateStackName(namespace, common.StackTypeConsul, environmentName)
		err := stackDeleter.DeleteStack(envStackName)
		if err != nil {
			return err
		}

		stack := stackWaiter.AwaitFinalStatus(envStackName)
		if stack != nil && !strings.HasSuffix(stack.Status, "_COMPLETE") {
			return fmt.Errorf("Ended in failed status %s %s", stack.Status, stack.StatusReason)
		}

		return nil
	}
}
func (workflow *environmentWorkflow) environmentEcsTerminator(namespace string, environmentName string, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter) Executor {
	return func() error {
		log.Noticef("Terminating ECS environment '%s' ...", environmentName)
		envStackName := common.CreateStackName(namespace, common.StackTypeEnv, environmentName)
		err := stackDeleter.DeleteStack(envStackName)
		if err != nil {
			return err
		}

		stack := stackWaiter.AwaitFinalStatus(envStackName)
		if stack != nil && !strings.HasSuffix(stack.Status, "_COMPLETE") {
			return fmt.Errorf("Ended in failed status %s %s", stack.Status, stack.StatusReason)
		}

		return nil
	}
}

func (workflow *environmentWorkflow) environmentRolesetTerminator(rolesetDeleter common.RolesetDeleter, environmentName string) Executor {
	return func() error {
		err := rolesetDeleter.DeleteEnvironmentRoleset(environmentName)
		if err != nil {
			return err
		}
		return nil
	}
}

func (workflow *environmentWorkflow) environmentElbTerminator(namespace string, environmentName string, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter) Executor {
	return func() error {
		log.Noticef("Terminating ELB environment '%s' ...", environmentName)
		envStackName := common.CreateStackName(namespace, common.StackTypeLoadBalancer, environmentName)
		err := stackDeleter.DeleteStack(envStackName)
		if err != nil {
			return err
		}

		stack := stackWaiter.AwaitFinalStatus(envStackName)
		if stack != nil && !strings.HasSuffix(stack.Status, "_COMPLETE") {
			return fmt.Errorf("Ended in failed status %s %s", stack.Status, stack.StatusReason)
		}

		return nil
	}
}
func (workflow *environmentWorkflow) environmentVpcTerminator(namespace string, environmentName string, stackDeleter common.StackDeleter, stackWaiter common.StackWaiter) Executor {
	return func() error {
		log.Noticef("Terminating VPC environment '%s' ...", environmentName)
		vpcStackName := common.CreateStackName(namespace, common.StackTypeVpc, environmentName)
		err := stackDeleter.DeleteStack(vpcStackName)
		if err != nil {
			log.Debugf("Unable to delete VPC, but ignoring error: %v", err)
		}

		stack := stackWaiter.AwaitFinalStatus(vpcStackName)
		if stack != nil && !strings.HasSuffix(stack.Status, "_COMPLETE") {
			return fmt.Errorf("Ended in failed status %s %s", stack.Status, stack.StatusReason)
		}

		targetStackName := common.CreateStackName(namespace, common.StackTypeTarget, environmentName)
		err = stackDeleter.DeleteStack(targetStackName)
		if err != nil {
			log.Debugf("Unable to delete VPC target, but ignoring error: %v", err)
		}

		stack = stackWaiter.AwaitFinalStatus(targetStackName)
		if stack != nil && !strings.HasSuffix(stack.Status, "_COMPLETE") {
			return fmt.Errorf("Ended in failed status %s %s", stack.Status, stack.StatusReason)
		}

		return nil
	}
}
