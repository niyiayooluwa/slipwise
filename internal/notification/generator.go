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

// Match Started — all use %s %s so Sprintf is safe
var startCopies = []string{
	"Game on! ⚽ %s vs %s just kicked off. We dey watch am for you.",
	"%s 🆚 %s — ball don roll. Settle in.",
	"Kickoff! %s vs %s just went live — your slip is in play 👀",
}

// HT Copies — all use %s vs %s so the match name is always shown
var htWinCopies = []string{
	"HT: %s vs %s dey shape well for you 🙌 Second half go seal am.",
	"Halftime — you're on the right side of this one (%s vs %s). No relax yet.",
}
var htLossCopies = []string{
	"HT: %s vs %s not looking great right now, but 45 minutes is a lot of football 🤞",
	"We need a massive second half in %s vs %s. Football is undefeated — anything fit happen.",
}
var htLevelCopies = []string{
	"HT: %s vs %s — score no move your market yet. Second half go decide am.",
}

// Early Hit — all entries use exactly two %s: selection, matchDesc
var hitCopies = []string{
	"Gett in jhoor! 🔥 %s don land for %s.",
	"Booooooooom!!!!!!!! 💥 %s just hit in %s.",
	"LFGGGGG!!!!!! 🚀 %s secured in %s (game still dey play).",
	"Odogwu behaviour — %s just ticked in %s 💰",
}

// VAR — all use %s %s
var varCopies = []string{
	"VAR strikes! 🚨 A goal just got ruled out in %s vs %s.",
	"Hold your celebration — VAR say no goal in %s vs %s. Back to how e be before.",
	"Ref changed him mind 🧐 That goal don cancel in %s vs %s.",
}

// Leg Lost — plain strings, no format verbs. Code goes in the title.
var legLostCopies = []string{
	"Damn. That one no gree work out. We go again.",
	"Ticket cut ❌ No wahala, next slip go pain them.",
	"This one pain small, but no shaking. Next!",
	"Ah omo. Football can be wicked like that. Bounce back.",
}

// Ticket Won — all use %d for legs count
var ticketWonCopies = []string{
	"TICKET DON PAY 🎉🎉 All %d legs landed. Odogwu behaviour.",
	"ALL %d LEGS LANDED. This one na testimony 🙌",
	"Booooooom, all %d legs cleared! Money dey your way 💰",
}

func GetStartMessage(home, away string) (title, body, image string) {
	title = "Game On! ⚽"
	body = fmt.Sprintf(startCopies[rand.Intn(len(startCopies))], home, away)
	image = startGifs[rand.Intn(len(startGifs))]
	return
}

// GetHTMessage now takes the match names so the body always shows which game it's about.
func GetHTMessage(home, away, status string) (title, body, image string) {
	title = "Halftime Check-in ⏱️"
	switch status {
	case "WON":
		body = fmt.Sprintf(htWinCopies[rand.Intn(len(htWinCopies))], home, away)
	case "LOST":
		body = fmt.Sprintf(htLossCopies[rand.Intn(len(htLossCopies))], home, away)
	default:
		body = fmt.Sprintf(htLevelCopies[rand.Intn(len(htLevelCopies))], home, away)
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

// GetTicketLossMessage — code goes in the title, body is a plain string.
func GetTicketLossMessage(code string) (title, body, image string) {
	title = fmt.Sprintf("Ticket %s cut ❌", code)
	body = legLostCopies[rand.Intn(len(legLostCopies))]
	image = cutGifs[rand.Intn(len(cutGifs))]
	return
}

// GetTicketWinMessage uses %d for total legs count.
func GetTicketWinMessage(legs int) (title, body, image string) {
	title = "BOOOOM! 🎉"
	body = fmt.Sprintf(ticketWonCopies[rand.Intn(len(ticketWonCopies))], legs)
	image = fwGifs[rand.Intn(len(fwGifs))]
	return
}
