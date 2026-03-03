package bot

import "github.com/bwmarrin/discordgo"

func HandleInteraction(s *discordgo.Session, ic *discordgo.InteractionCreate) {
	if ic.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch ic.ApplicationCommandData().Name {
	case "ping":
		respond(s, ic, "pong", false)
	case "help":
		handleHelp(s, ic)
	case "join":
		handleJoin(s, ic)
	case "leave":
		handleLeave(s, ic)
	case "list":
		handleList(s, ic)
	case "make":
		handleMake(s, ic)
	case "next":
		handleNext(s, ic)
	case "pause":
		handlePause(s, ic)
	case "resume":
		handleResume(s, ic)
	case "paused":
		handlePaused(s, ic)
	case "whoami":
		handleWhoAmI(s, ic)
	case "undo":
		handleUndo(s, ic)
	case "reroll":
		handleReroll(s, ic)
	case "reset":
		handleReset(s, ic)
	case "result":
		handleResult(s, ic)
	case "export":
		handleExport(s, ic)
	case "settings":
		handleSettings(s, ic)
	}
}
