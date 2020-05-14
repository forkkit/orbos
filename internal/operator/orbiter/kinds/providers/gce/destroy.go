package gce

func destroy(desired *Spec, context *context) error {

	if err := chain(
		context, nil,
		queryForwardingRules,
		queryAddresses,
		queryTargetPools,
		queryHealthchecks,
		queryFirewall,
	); err != nil {
		return err
	}

	pools, err := context.machinesService.ListPools()
	if err != nil {
		return err
	}
	for _, pool := range pools {
		machines, err := context.machinesService.List(pool)
		if err != nil {
			return err
		}
		for _, machine := range machines {
			if err := machine.Remove(); err != nil {
				return err
			}
		}
	}
	desired.SSHKey = nil
	return nil
}
