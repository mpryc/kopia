package cli

type commandRepository struct {
	connect          commandRepositoryConnect
	create           commandRepositoryCreate
	disconnect       commandRepositoryDisconnect
	repair           commandRepositoryRepair
	setClient        commandRepositorySetClient
	setParameters    commandRepositorySetParameters
	changePassword   commandRepositoryChangePassword
	status           commandRepositoryStatus
	syncTo           commandRepositorySyncTo
	throttle         commandRepositoryThrottle
	validateProvider commandRepositoryValidateProvider
	upgrade          commandRepositoryUpgrade
}

func (c *commandRepository) setup(svc advancedAppServices, parent commandParent) {
	cmd := parent.Command("bslserver", "Commands to connect repository.").Alias("bsls")

	c.connect.setup(svc, cmd)
	c.disconnect.setup(svc, cmd)
	c.status.setup(svc, cmd)
}
