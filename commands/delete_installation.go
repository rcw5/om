package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/pivotal-cf/jhanda/commands"
	"github.com/pivotal-cf/om/api"
)

type DeleteInstallation struct {
	deleteService        installationAssetDeleterService
	installationsService installationsService
	logger               logger
	logWriter            logWriter
	waitDuration         int
}

//go:generate counterfeiter -o ./fakes/installation_asset_deleter_service.go --fake-name InstallationAssetDeleterService . installationAssetDeleterService
type installationAssetDeleterService interface {
	Delete() (api.InstallationsServiceOutput, error)
}

func NewDeleteInstallation(deleteService installationAssetDeleterService, installationsService installationsService, logWriter logWriter, logger logger, waitDuration int) DeleteInstallation {
	return DeleteInstallation{
		deleteService:        deleteService,
		installationsService: installationsService,
		logger:               logger,
		logWriter:            logWriter,
		waitDuration:         waitDuration,
	}
}

func (ac DeleteInstallation) Execute(args []string) error {
	installation, err := ac.installationsService.RunningInstallation()

	if installation == (api.InstallationsServiceOutput{}) {
		ac.logger.Printf("attempting to delete the installation on the targeted Ops Manager")

		installation, err = ac.deleteService.Delete()
		if err != nil {
			return fmt.Errorf("failed to delete installation: %s", err)
		}

		if installation == (api.InstallationsServiceOutput{}) {
			ac.logger.Printf("no installation to delete")
			return nil
		}
	} else {
		ac.logger.Printf("found already running deletion...attempting to re-attach")
	}

	for {
		current, err := ac.installationsService.Status(installation.ID)
		if err != nil {
			return fmt.Errorf("installation failed to get status: %s", err)
		}

		install, err := ac.installationsService.Logs(installation.ID)
		if err != nil {
			return fmt.Errorf("installation failed to get logs: %s", err)
		}

		err = ac.logWriter.Flush(install.Logs)
		if err != nil {
			return fmt.Errorf("installation failed to flush logs: %s", err)
		}

		if current.Status == api.StatusSucceeded {
			return nil
		} else if current.Status == api.StatusFailed {
			return errors.New("deleting the installation was unsuccessful")
		}

		time.Sleep(time.Duration(ac.waitDuration) * time.Second)
	}
}

func (ac DeleteInstallation) Usage() commands.Usage {
	return commands.Usage{
		Description:      "This authenticated command deletes all the products installed on the targeted Ops Manager.",
		ShortDescription: "deletes all the products on the Ops Manager targeted",
	}
}
