package gna

/*Instance is your game state. Each method runs concurrently with one another.
You're meant to provide Auth, Disconn and Update only and let NetAbs and Terminate
be used from the embedded Net struct
*/
type Instance interface {
	/*Update loop of your game state, you can get the batch of data from the Players
	with Instance.GetData(), and dispatch it with Instance.Dispatch*/
	Update()
	/*Disconn happens when a player disconnects*/
	Disconn(*Player)
	/*Auth happens when a player connects, to refuse the player connection
	simply close it: Player.Close(). To accept it, leave it be. The instance is
	the main instance of the server, if you do not manually set the instance with
	Player.SetInstance, the player is set to the main instance.
	*/
	Auth(*Player)
	/*NetAbs exposes the underlying networking abstraction,
	  this is for internal use only.*/
	NetAbs() *Net
	/*Terminate closes all connections inside the instance and stops the updates*/
	Terminate()
}

/*RunInstance starts the ticker and Disconnection Handler,
it's the only place where Instance.Update is called.
If RunInstance is called twice in a Instance it just returns.
*/
func RunInstance(ins Instance) {
	n := ins.NetAbs()
	if n.started {
		return
	}
	n.fillDefault()
	go dcHandler(n.dc, ins)
	n.started = true
	for {
		select {
		case <-n.ticker.C:
			ins.Update()
		case <-n.done:
			return
		}
	}
}

func dcHandler(dc chan *Player, ins Instance) {
	n := ins.NetAbs()
	for {
		p := <-dc
		n.Players.Rm(p.ID)
		ins.Disconn(p)
	}
}
