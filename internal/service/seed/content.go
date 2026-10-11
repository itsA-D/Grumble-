package seed

import "grumble/internal/domain/pain"

// Item is one piece of labelled seed content. Category and Country are catalog
// slugs/codes resolved against the taxonomy when the seed runs; Votes is the
// number of synthetic votes to attach so the feed, hubs and ranking look
// populated. Seed rows are always stored with seeded = true and are never
// presented as organic.
type Item struct {
	Body          string
	Category      string
	Country       string
	Frequency     pain.Frequency
	Disruption    pain.Disruption
	WouldPay      bool
	MonthlyAmount int
	Tried         string
	Votes         int
}

// content is the labelled seed set. Bodies are original plain text spread across
// categories and countries; none contains personal data.
var content = []Item{
	// business (5)
	{Body: "Team stand-ups turn into forty-minute status theatre every single morning", Category: "business", Country: "US", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 15, Votes: 34},
	{Body: "Nobody knows who actually owns the shared inbox and real requests go missing", Category: "business", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 25, Votes: 21},
	{Body: "We keep reopening the same decision every quarter because nothing is written down", Category: "business", Country: "DE", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 12},
	{Body: "Expense reports need six approvals for a ten dollar lunch", Category: "business", Country: "CA", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 8},
	{Body: "Meeting invites arrive with no agenda and a full hour blocked", Category: "business", Country: "AU", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 10, Votes: 17},

	// health (3)
	{Body: "Getting a routine appointment takes three weeks and two phone calls", Category: "health", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, Votes: 45},
	{Body: "Every clinic wants the same forms filled in again on paper", Category: "health", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 20, Votes: 26},
	{Body: "Prescription refills still require physically visiting the pharmacy", Category: "health", Country: "CA", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, Votes: 14},

	// education (3)
	{Body: "Parents get grades and homework lists through five different apps", Category: "education", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 8, Votes: 19},
	{Body: "Enrollment forms are lost between departments every single year", Category: "education", Country: "IN", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, Votes: 22},
	{Body: "Course materials are scattered across portals nobody can search", Category: "education", Country: "AU", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionMild, Votes: 9},

	// housing (5)
	{Body: "Renters wait weeks for a landlord to fix a broken heater in winter", Category: "housing", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 30, Votes: 41},
	{Body: "Viewing a flat means competing with thirty people in a ten minute slot", Category: "housing", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, Votes: 33},
	{Body: "Deposits are returned months late with no itemised deduction", Category: "housing", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 18, Votes: 27},
	{Body: "Building management never answers the intercom or the phone", Category: "housing", Country: "FR", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, Votes: 11},
	{Body: "Moving utilities to a new address takes five calls and a final bill mix-up", Category: "housing", Country: "NL", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 13},

	// finance (3)
	{Body: "Bank fraud alerts freeze the card then ask you to call during office hours", Category: "finance", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 22, Votes: 29},
	{Body: "Switching a direct debit between banks takes a full billing cycle", Category: "finance", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 16},
	{Body: "Every subscription needs a support chat to cancel, never a button", Category: "finance", Country: "DE", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 12, Votes: 35},

	// travel (3)
	{Body: "Flight changes are quoted at triple the price of booking a new seat", Category: "travel", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 40, Votes: 24},
	{Body: "Airport transfers never align with the actual arrival time of the plane", Category: "travel", Country: "ES", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 10},
	{Body: "Visa paperwork repeats the same questions on four separate forms", Category: "travel", Country: "IN", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, Votes: 18},

	// tech (6)
	{Body: "Password resets take twenty minutes and land in the spam folder anyway", Category: "tech", Country: "US", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 9, Votes: 52},
	{Body: "Software updates restart the machine in the middle of a call", Category: "tech", Country: "GB", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 14, Votes: 38},
	{Body: "Two-factor prompts arrive minutes after the code has expired", Category: "tech", Country: "CA", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, Votes: 30},
	{Body: "Support tickets vanish with no reference number and no reply", Category: "tech", Country: "DE", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 20},
	{Body: "The same notification is duplicated across every device you own", Category: "tech", Country: "SG", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionMild, Votes: 15},
	{Body: "Free trials renew silently because cancellation is buried in settings", Category: "tech", Country: "AU", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 11, Votes: 25},

	// relationships (2)
	{Body: "Group chats split across three apps so plans are always lost", Category: "relationships", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 12},
	{Body: "Scheduling anything with friends takes a week of back and forth", Category: "relationships", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 14},

	// food (3)
	{Body: "Grocery delivery substitutions are random and never refunded", Category: "food", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 7, Votes: 23},
	{Body: "Restaurant bookings only open by phone at exactly ten in the morning", Category: "food", Country: "IT", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 11},
	{Body: "Menu prices differ between the window, the app and the receipt", Category: "food", Country: "FR", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 8},

	// government (5)
	{Body: "Renewing a licence means printing, posting and waiting six weeks", Category: "government", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 16, Votes: 37},
	{Body: "Two departments ask for the same proof of address in different formats", Category: "government", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 22},
	{Body: "The online portal is down whenever you have time to use it", Category: "government", Country: "IN", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 19},
	{Body: "Tax forms change every year with no summary of what actually changed", Category: "government", Country: "DE", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 21, Votes: 27},
	{Body: "Public records live in a dozen incompatible websites", Category: "government", Country: "BR", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 9},

	// sports (2)
	{Body: "Match tickets sell out to bots before any fan can log in", Category: "sports", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 13, Votes: 20},
	{Body: "League standings are published a day late on the official site", Category: "sports", Country: "ES", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 6},

	// entertainment (3)
	{Body: "Seasons are split across four streaming services at once", Category: "entertainment", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 17, Votes: 32},
	{Body: "Cinema seats are sold as available then double-booked at the door", Category: "entertainment", Country: "CA", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 10},
	{Body: "Concert queues crash the moment tickets go on sale", Category: "entertainment", Country: "AU", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, Votes: 18},

	// gaming (3)
	{Body: "Every launcher wants its own account, friends list and updater", Category: "gaming", Country: "US", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 6, Votes: 28},
	{Body: "Matchmaking pairs you with players a whole skill tier away", Category: "gaming", Country: "KR", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, Votes: 16},
	{Body: "Save data is trapped on one console generation", Category: "gaming", Country: "JP", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 7},

	// parenting (2)
	{Body: "School closures are announced by text at seven in the morning", Category: "parenting", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 19, Votes: 21},
	{Body: "Childcare waitlists have no position and no estimated date", Category: "parenting", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, Votes: 25},

	// pets (2)
	{Body: "Emergency vets charge a deposit before they will even look at the pet", Category: "pets", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 24, Votes: 17},
	{Body: "Pet registration renewals are annual paperwork with no reminder", Category: "pets", Country: "AU", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 5},

	// fitness (2)
	{Body: "Gym memberships renew annually and cancel only by recorded letter", Category: "fitness", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 15, Votes: 26},
	{Body: "Class timetables change weekly and bookings open at odd hours", Category: "fitness", Country: "DE", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 8},

	// fashion (2)
	{Body: "Online sizes vary so every order becomes a return", Category: "fashion", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 12, Votes: 20},
	{Body: "Returns need a printer and a drop-off point that closes at five", Category: "fashion", Country: "FR", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 11},

	// transportation (5)
	{Body: "The train is late every single morning and the refund form is hidden", Category: "transportation", Country: "GB", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 20, Votes: 48},
	{Body: "Bus timetables are fiction and the live tracker is wrong too", Category: "transportation", Country: "IE", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, Votes: 23},
	{Body: "Parking apps charge a fee on top of the parking itself", Category: "transportation", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 13},
	{Body: "Bike lanes end abruptly and dump you into traffic", Category: "transportation", Country: "NL", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionRuinsWeek, Votes: 19},
	{Body: "Roadworks signs stay up for months after the work finishes", Category: "transportation", Country: "DE", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 9},

	// environment (2)
	{Body: "Recycling rules differ by street and nobody can find the actual list", Category: "environment", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 12},
	{Body: "Bottle deposit machines are always full or out of order", Category: "environment", Country: "DE", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 10},

	// career (3)
	{Body: "Job applications ask you to retype the entire CV into fields", Category: "career", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 13, Votes: 31},
	{Body: "Recruiters go silent for weeks then demand an interview tomorrow", Category: "career", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 24},
	{Body: "Internal promotions are decided before the role is even posted", Category: "career", Country: "IN", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionRuinsWeek, Votes: 18},

	// shopping (5)
	{Body: "Packaging is three boxes for one tiny item and the bin is already full", Category: "shopping", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 14},
	{Body: "Click and collect is never ready when the app says it is", Category: "shopping", Country: "GB", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 19},
	{Body: "Loyalty points expire the week before you plan to use them", Category: "shopping", Country: "CA", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, WouldPay: true, MonthlyAmount: 8, Votes: 16},
	{Body: "Price tags at the till never match the shelf", Category: "shopping", Country: "AU", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 11},
	{Body: "Support only exists as a chatbot that loops back to the start", Category: "shopping", Country: "SG", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionAnnoying, Votes: 22},

	// other (2)
	{Body: "Every form asks for a fax number in the year of our lord two thousand and twenty six", Category: "other", Country: "US", Frequency: pain.FrequencyWeekly, Disruption: pain.DisruptionMild, Votes: 27},
	{Body: "Captcha puzzles get harder the more carefully you solve them", Category: "other", Country: "US", Frequency: pain.FrequencyDaily, Disruption: pain.DisruptionAnnoying, Votes: 33},
}
