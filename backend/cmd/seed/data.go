package main

// Словарные банки для генерации правдоподобных названий/текстов.

var firstNames = []string{
	"Alex", "Maria", "John", "Olga", "David", "Anna", "Michael", "Elena", "Chris", "Nina",
	"Daniel", "Sofia", "James", "Kate", "Robert", "Vera", "Tom", "Lena", "Mark", "Julia",
	"Peter", "Dasha", "Sergey", "Emma", "Pavel", "Lucy", "Ivan", "Mia", "Andrew", "Zoe",
}

var lastNames = []string{
	"Smith", "Ivanov", "Brown", "Petrov", "Johnson", "Sidorov", "Lee", "Kuznetsov", "Garcia",
	"Volkov", "Miller", "Sokolov", "Davis", "Popov", "Wilson", "Orlov", "Moore", "Lebedev",
	"Taylor", "Novak", "Clark", "Roman", "Walker", "Frost", "Young", "Stone", "Hall", "Reed",
}

var teamKeys = []string{"ENG", "DES", "OPS", "QA", "DATA", "PLAT", "WEB", "MOB", "INFRA", "GROW"}
var teamNames = []string{
	"Engineering", "Design", "Operations", "Quality", "Data", "Platform",
	"Web", "Mobile", "Infrastructure", "Growth",
}

var projectNames = []string{"Apollo", "Mercury", "Atlas", "Phoenix", "Nova", "Orion", "Titan", "Vega"}
var projectSuffix = []string{"Migration", "Revamp", "Launch", "Rollout", "Initiative", "Overhaul"}

var milestoneNames = []string{
	"v1.0 GA", "Beta", "Public Launch", "Q3 Release", "MVP", "Hardening",
	"Performance Pass", "Security Audit", "Internal Preview", "Q4 Cut",
}

var viewNames = []string{
	"My open issues", "High priority", "This cycle", "Backlog grooming", "Recently updated",
	"Bugs", "Needs review", "Blocked", "Roadmap", "Unassigned", "Done this week",
}

var channelNames = []string{
	"general", "random", "engineering", "design", "product", "incidents", "releases",
	"support", "standup", "watercooler", "hiring", "sales", "marketing", "security",
	"infra", "frontend", "backend", "data", "qa", "announcements",
}
var channelSuffix = []string{"team", "chat", "sync", "hq", "room"}

var docTitles = []string{
	"Onboarding guide", "Architecture overview", "API conventions", "Release process",
	"Incident runbook", "Coding standards", "RFC: caching layer", "Postmortem 2026-05",
	"Team charter", "Roadmap notes", "Design system", "Security policy", "On-call rotation",
	"Deployment checklist", "Glossary", "Meeting notes", "Q3 planning", "Tech debt register",
}
var docIcons = []string{"📄", "📘", "🗺️", "🔧", "🚀", "🔐", "📊", "📝", "📌"}

var meetingTitles = []string{
	"Daily standup", "Sprint planning", "Retro", "1:1", "Design review", "Architecture sync",
	"Incident review", "Roadmap planning", "Demo", "All-hands", "Backlog grooming", "Sync",
}

var calTitles = []string{
	"Focus time", "Standup", "1:1", "Design review", "Planning", "Code review",
	"Customer call", "Interview", "Lunch", "Gym", "Deep work", "Sync", "Demo prep",
}
var dayoffTitles = []string{"Vacation", "Sick day", "PTO", "Holiday", "Day off"}

var timeDescs = []string{
	"Implementation", "Debugging", "Code review", "Investigation", "Pairing",
	"Testing", "Refactor", "Research", "Meeting", "Writing docs",
}

var emojis = []string{"👍", "🎉", "🔥", "👀", "✅", "❤️", "🚀", "😄", "🙏", "💯"}

var verbs = []string{
	"Fix", "Add", "Refactor", "Improve", "Implement", "Update", "Remove", "Investigate",
	"Optimize", "Migrate", "Document", "Redesign", "Support", "Handle", "Prevent",
}
var nouns = []string{
	"login flow", "search index", "cache layer", "rate limiter", "webhook", "dashboard",
	"export", "notification", "permission check", "retry logic", "pagination", "filter",
	"upload", "sync", "session", "queue", "migration", "validation", "tooltip", "modal",
}
var areas = []string{
	"on mobile", "in the API", "for admins", "under load", "for SSO", "in the editor",
	"on cold start", "across tenants", "in dark mode", "for large workspaces", "", "", "",
}

var loremWords = []string{
	"the", "team", "should", "consider", "this", "change", "before", "release", "since",
	"it", "affects", "performance", "and", "user", "experience", "across", "all", "tenants",
	"we", "need", "to", "verify", "behavior", "under", "load", "with", "proper", "metrics",
	"in", "place", "for", "observability", "let's", "ship", "a", "small", "fix", "first",
	"then", "iterate", "based", "on", "feedback", "from", "early", "adopters", "next", "week",
}

var chatLines = []string{
	"lgtm 👍", "shipping now", "can someone review this?", "on it", "fixed in latest",
	"heads up: deploying in 5", "nice work!", "what's the status here?", "blocked on infra",
	"let's sync tomorrow", "good catch", "merged", "tests are green", "rolling back",
	"PTAL when you get a sec", "+1", "done", "thanks!", "any update?", "looking into it",
}

var changeFields = []string{"state_id", "assignee_id", "priority", "title", "estimate", "due_date", "cycle_id", "milestone_id"}

var autoNames = []string{
	"Auto-assign new bugs", "Close stale issues", "Triage incoming", "Notify on done",
	"Label hotfixes", "Set priority on creation", "Assign to lead", "Move to current cycle",
}

var dashNames = []string{
	"Team overview", "Sprint health", "Bug triage", "Roadmap", "Exec summary", "QA status",
}

var widgetTitles = []string{
	"Open issues", "By priority", "Velocity", "Recent activity", "Backlog size",
	"Status breakdown", "Notes", "This cycle", "Throughput",
}

var cfDefs = []cfDef{
	{"severity", "Severity", "select", []string{"S1", "S2", "S3", "S4"}},
	{"impact", "Impact", "select", []string{"Low", "Medium", "High"}},
	{"pod", "Pod", "select", []string{"Alpha", "Bravo", "Charlie", "Delta"}},
	{"customer", "Customer", "text", nil},
	{"reviewers", "Reviewers", "number", nil},
	{"env", "Environment", "select", []string{"dev", "staging", "prod"}},
}

var labelDefs = []labelDef{
	{"bug", "#f7768e"},
	{"feature", "#9ece6a"},
	{"enhancement", "#7aa2f7"},
	{"docs", "#7dcfff"},
	{"design", "#bb9af7"},
	{"backend", "#e0af68"},
	{"frontend", "#ff9e64"},
	{"infra", "#73daca"},
	{"security", "#f7768e"},
	{"tech-debt", "#565f89"},
	{"good-first-issue", "#9ece6a"},
	{"urgent", "#db4b4b"},
	{"blocked", "#ff9e64"},
	{"needs-review", "#7aa2f7"},
}
