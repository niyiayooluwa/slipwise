package notification

import (
	"fmt"
	"math/rand"
)

// Gifs
var startGifs = []string{"https://media.giphy.com/media/l0HlJDaeqNpbxRtwI/giphy.gif"} // whistle
var htGifs = []string{"https://media.giphy.com/media/l41YwV97F0iY6Wv2g/giphy.gif"}    // clock
var hitGifs = []string{"https://media.giphy.com/media/3o7TKoWXm3okO1kgHC/giphy.gif"}  // champagne
var varGifs = []string{"https://media.giphy.com/media/1rSpWBMQlQJzjglX2I/giphy.gif"}  // VAR
var cutGifs = []string{"https://media.giphy.com/media/d2lcHJTG5Tscg/giphy.gif"}       // heartbreak
var fwGifs = []string{"https://media.giphy.com/media/26tOZ42Mg6pbTUPHW/giphy.gif"}    // fireworks

// Match Started
var startCopies = []string{
	"Game on! ⚽ %s vs %s just kicked off. We dey watch am for you.",
	"%s 🆚 %s — ball don roll. Settle in.",
	"Kickoff! Your slip just went live on this one 👀",
}

// HT Copies
var htWinCopies = []string{
	"HT: You dey lead so far 🙌 Second half go still make sense.",
	"Halftime and things dey shape well for you. No relax yet.",
}
var htLossCopies = []string{
	"HT: Not looking great right now, but 45 minutes is a lot of football 🤞",
	"We need a massive second half. Football is undefeated though — anything fit happen.",
}
var htLevelCopies = []string{
	"HT: Score no move your market yet. Second half go decide am.",
}

// Early Hit
var hitCopies = []string{
	"Gett in jhoor! 🔥 %s don land for %s.",
	"Booooooooom!!!!!!!! 💥 %s is IN.",
	"Odogwu! One leg just paid itself 💰",
	"LFGGGGG!!!!!! 🚀 %s secured (for now — game still dey play).",
}

// VAR
var varCopies = []string{
	"VAR strikes! 🚨 A goal just got ruled out in %s vs %s.",
	"Hold your celebration — VAR say no goal. Back to how e be before.",
	"Ref changed him mind 🧐 That goal don cancel.",
}

// Leg Lost
var legLostCopies = []string{
	"Ah omo, ticket don cut 💔 (Ticket %s)",
	"Damn. That one no gree work out. We go again.",
	"Ticket cut ❌ No wahala, next slip go pain them.",
	"This one pain small, but no shaking. Next!",
}

// Ticket Won
var ticketWonCopies = []string{
	"TICKET DON PAY 🎉🎉 Odogwu behaviour.",
	"ALL %d LEGS LANDED. This one na testimony 🙌",
	"Booooooom, full ticket cleared! 💰",
}

func GetStartMessage(home, away string) (title, body, image string) {
	title = "Game On! ⚽"
	body = fmt.Sprintf(startCopies[rand.Intn(len(startCopies))], home, away)
	image = startGifs[rand.Intn(len(startGifs))]
	return
}

func GetHTMessage(status string) (title, body, image string) {
	title = "Halftime Check-in ⏱️"
	if status == "WON" {
		body = htWinCopies[rand.Intn(len(htWinCopies))]
	} else if status == "LOST" {
		body = htLossCopies[rand.Intn(len(htLossCopies))]
	} else {
		body = htLevelCopies[rand.Intn(len(htLevelCopies))]
	}
	image = htGifs[rand.Intn(len(htGifs))]
	return
}

func GetEarlyHitMessage(selection, matchDesc string) (title, body, image string) {
	title = "Early Hit 🔥"
	body = fmt.Sprintf(hitCopies[rand.Intn(len(hitCopies))], selection, matchDesc)
	image = hitGifs[rand.Intn(len(hitGifs))]
	return
}

func GetVARMessage(home, away string) (title, body, image string) {
	title = "VAR Alert 🚨"
	body = fmt.Sprintf(varCopies[rand.Intn(len(varCopies))], home, away)
	image = varGifs[rand.Intn(len(varGifs))]
	return
}

func GetTicketLossMessage(code string) (title, body, image string) {
	title = "Ticket Busted ❌"
	body = fmt.Sprintf(legLostCopies[rand.Intn(len(legLostCopies))], code)
	image = cutGifs[rand.Intn(len(cutGifs))]
	return
}

func GetTicketWinMessage(legs int, code string) (title, body, image string) {
	title = "BOOOOM! 🎉"
	body = fmt.Sprintf(ticketWonCopies[rand.Intn(len(ticketWonCopies))], legs)
	image = fwGifs[rand.Intn(len(fwGifs))]
	return
}
