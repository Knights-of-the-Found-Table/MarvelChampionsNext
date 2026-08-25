// Package storm registers Storm, her Weather deck, signature cards, obligation, and nemesis set.
package storm

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

const (
	weatherClearSignal     = -36002
	weatherHurricaneSignal = -36003
	weatherThunderSignal   = -36004
	weatherBlizzardSignal  = -36005
	blankMinionSignal      = -36050
	refreshWeatherSignal   = -36090
)

var weatherCodes = []string{"36002", "36003", "36004", "36005"}

func init() {
	registerStorm()
	registerWeatherDeck()
	registerStormSignatures()
	registerStormObligation()
	registerStormNemesis()
}

func weatherSignal(code string) int {
	switch code {
	case "36002":
		return weatherClearSignal
	case "36003":
		return weatherHurricaneSignal
	case "36004":
		return weatherThunderSignal
	case "36005":
		return weatherBlizzardSignal
	}
	return 0
}

func signalWeather(n int) string {
	switch n {
	case weatherClearSignal:
		return "36002"
	case weatherHurricaneSignal:
		return "36003"
	case weatherThunderSignal:
		return "36004"
	case weatherBlizzardSignal:
		return "36005"
	}
	return ""
}

func currentWeather(g *engine.Game, p *engine.Player) *engine.Support {
	if p == nil {
		return nil
	}
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil {
			for _, code := range weatherCodes {
				if s.Code == code {
					return s
				}
			}
		}
	}
	return nil
}

func weatherChoices(g *engine.Game, p *engine.Player, prompt string) []engine.Message {
	if p == nil {
		return nil
	}
	current := currentWeather(g, p)
	choices := make([]engine.Choice, 0, len(weatherCodes))
	for _, code := range weatherCodes {
		if current != nil && current.Code == code {
			continue
		}
		def := engine.DB.MustLookup(code)
		choices = append(choices, engine.Choice{
			Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: code,
		}.Msgs(engine.AddEntityCounter{ID: p.ID, N: weatherSignal(code)}))
	}
	if len(choices) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.S(prompt), choices...)}}
}

func clearWeatherStatuses(g *engine.Game) []engine.Message {
	var msgs []engine.Message
	for _, p := range g.Players {
		msgs = append(msgs, engine.ClearStun{Target: p.ID}, engine.ClearConfuse{Target: p.ID})
		for _, id := range p.Allies {
			msgs = append(msgs, engine.ClearStun{Target: id}, engine.ClearConfuse{Target: id})
		}
	}
	for _, id := range cardutil.SortedIDs(g.Villains) {
		msgs = append(msgs, engine.ClearStun{Target: id}, engine.ClearConfuse{Target: id})
	}
	for _, id := range cardutil.SortedIDs(g.Minions) {
		msgs = append(msgs, engine.ClearStun{Target: id}, engine.ClearConfuse{Target: id})
	}
	return msgs
}

func weatherAttackDelta(code string) int {
	switch code {
	case "36004":
		return 1
	case "36005":
		return -1
	default:
		return 0
	}
}

func adjustWeatherAttack(g *engine.Game, code string, direction int) {
	delta := weatherAttackDelta(code) * direction
	if delta == 0 {
		return
	}
	for _, p := range g.Players {
		p.BonusATK += delta
	}
	// Ally attack values are captured when their action menu is built, and
	// there is no global-stat hook for supports. Avoid corrupting allies that
	// enter after a Weather swap; their +1/-1 ATK is the one omitted aura case.
	for _, v := range g.Villains {
		v.AttackVal += delta
	}
	for _, mn := range g.Minions {
		mn.AttackVal += delta
	}
}

func stormCapeMessages(g *engine.Game, p *engine.Player) []engine.Message {
	if p == nil {
		return nil
	}
	for _, id := range p.Upgrades {
		if cape := g.Upgrades[id]; cape != nil && cape.Code == "36007" && !cape.Exhausted {
			// Named Special resolutions do not expose an optional-response window,
			// so Storm's Cape is resolved automatically when it is ready.
			return []engine.Message{engine.ExhaustEntity{ID: cape.ID}, engine.ReadyEntity{ID: p.ID}}
		}
	}
	return nil
}

func weatherSpecial(g *engine.Game, p *engine.Player) []engine.Message {
	weather := currentWeather(g, p)
	if weather == nil {
		return nil
	}
	var msgs []engine.Message
	switch weather.Code {
	case "36002":
		msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
	case "36003":
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.ThwartScheme{Scheme: id, N: 2, Source: p.ID}}
		})
		if len(choices) > 0 {
			msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.hurricaneChooseAScheme"), choices...)})
		}
	case "36004":
		choices := cardutil.EnemyChoices(g, 2, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}}
		})
		if len(choices) > 0 {
			msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.thunderstormChooseAnEnemy"), choices...)})
		}
	case "36005":
		var choices []engine.Choice
		for _, id := range cardutil.SortedIDs(g.Minions) {
			mn := g.Minions[id]
			if mn == nil || mn.EDef().HasTrait("elite") {
				continue
			}
			choices = append(choices, engine.Choice{
				Label: engine.S(mn.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: mn.Code,
			}.Msgs(engine.AddEntityCounter{ID: id, N: blankMinionSignal}))
		}
		if len(choices) > 0 {
			msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.blizzardChooseANonEliteMinion"), choices...)})
		}
	}
	return append(msgs, stormCapeMessages(g, p)...)
}

func swapWeather(g *engine.Game, p *engine.Player, code string) []engine.Message {
	if p == nil || signalWeather(weatherSignal(code)) == "" {
		return nil
	}
	if old := currentWeather(g, p); old != nil {
		adjustWeatherAttack(g, old.Code, -1)
		g.Delete(old.ID)
	}
	s := &engine.Support{ID: g.NextEntityID("support"), Code: code, Owner: p.ID}
	g.Supports[s.ID] = s
	p.Supports = append(p.Supports, s.ID)
	adjustWeatherAttack(g, code, 1)
	g.TLogf("c.putsIntoPlayFromTheWeatherDeck", p.Name, s)
	msgs := []engine.Message{}
	if code == "36002" {
		msgs = append(msgs, clearWeatherStatuses(g)...)
	}
	return append(msgs, weatherSpecial(g, p)...)
}

func registerStorm() {
	engine.RegisterBehavior("36001", &engine.Behavior{
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			return weatherChoices(g, p, "Ororo Munroe — choose starting Weather")
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if p == nil || !p.IsHero() || currentWeather(g, p) == nil {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.weatherControlSwapWeatherAndResolveItsSpecial"),
				Type:  engine.AbilityAction, HeroOnly: true, OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return weatherChoices(g, g.Player(self), "Weather Control — choose Weather")
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			m, ok := msg.(engine.AddEntityCounter)
			if !ok {
				return nil
			}
			if m.ID == p.ID {
				if code := signalWeather(m.N); code != "" {
					return swapWeather(g, p, code)
				}
				if m.N == refreshWeatherSignal {
					if weather := currentWeather(g, p); weather != nil {
						p.BonusATK += weatherAttackDelta(weather.Code)
					}
				}
			}
			if m.N == blankMinionSignal {
				if mn := g.Minions[m.ID]; mn != nil && !mn.EDef().HasTrait("elite") {
					// BlankText expires at phase end in the engine, earlier than the
					// printed end-of-round duration.
					mn.BlankText = true
				}
			}
			return nil
		},
	})
}

func registerWeatherDeck() {
	engine.RegisterBehavior("36002", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		switch m := msg.(type) {
		case engine.StunEntity:
			return []engine.Message{engine.ClearStun{Target: m.Target}}
		case engine.ConfuseEntity:
			return []engine.Message{engine.ClearConfuse{Target: m.Target}}
		}
		return nil
	}})
	engine.RegisterBehavior("36003", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.WindowAfterEnemyAttacked)
		if !ok {
			return nil
		}
		// The attack window does not expose the actual defender. Treat every
		// completed enemy attack as defended for Hurricane's global Retaliate 1.
		return []engine.Message{engine.DamageEntity{Target: m.Enemy, Damage: 1, Source: e.EID()}}
	}})
	for _, code := range []string{"36004", "36005"} {
		engine.RegisterBehavior(code, &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.EndPhase:
				// Player BonusATK is phase-scoped. Reapply the persistent Weather
				// modifier after the engine clears temporary bonuses.
				return []engine.Message{engine.AddEntityCounter{ID: s.Owner, N: refreshWeatherSignal}}
			case engine.MinionEntersPlay:
				if mn := g.Minions[m.MinionID]; mn != nil {
					mn.AttackVal += weatherAttackDelta(s.Code)
				}
			case engine.AdvanceVillainStage:
				// Stage advancement refreshes printed ATK after reactions. The
				// current stage can therefore miss the Weather modifier until the
				// next swap; this is a known hook limitation.
			}
			return nil
		}})
	}
}

func registerStormSignatures() {
	engine.RegisterBehavior("36006", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
		// ResourceAbility cannot select its icon dynamically. Wild preserves
		// payment flexibility while approximating the printed Weather icon.
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})
	engine.RegisterBehavior("36007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{DEF: 1} },
	})
	engine.RegisterBehavior("36008", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: engine.Tf("c.ororoSGardenHeal2Damage"), Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				s := g.Supports[self]
				if s == nil {
					return nil
				}
				return []engine.Message{engine.HealEntity{Target: s.Owner, N: 2}}
			},
		}}
	}})
	engine.RegisterBehavior("36009", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		return weatherChoices(g, g.Player(e.EOwner()), "Weather Goddess — choose Weather")
	}})
	engine.RegisterBehavior("36010", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			msgs := []engine.Message{engine.ThwartScheme{Scheme: id, N: 3, Source: p.ID}}
			if weather := currentWeather(g, p); weather != nil && weather.Code == "36003" {
				msgs = append(msgs, weatherSpecial(g, p)...)
			}
			return msgs
		})
		if len(choices) == 0 {
			return nil
		}
		// The engine has no split-threat chooser, so all 3 threat is removed
		// from one selected scheme.
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.torrentialRainChooseAScheme"), choices...)}}
	}})
	engine.RegisterBehavior("36011", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		choices := cardutil.EnemyChoices(g, 8, p.ID, func(id engine.EntityID) []engine.Message {
			msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 8, Source: p.ID}}
			if weather := currentWeather(g, p); weather != nil && weather.Code == "36004" {
				msgs = append(msgs, weatherSpecial(g, p)...)
			}
			return msgs
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.lightningBoltChooseAnEnemy"), choices...)}}
	}})
	engine.RegisterBehavior("36012", &engine.Behavior{DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
		var msgs []engine.Message
		if weather := currentWeather(g, p); weather != nil && weather.Code == "36005" {
			msgs = append(msgs, weatherSpecial(g, p)...)
		}
		// ExtraPrevent models -3 ATK for the current villain attack. The
		// phase-long reduction to engaged minions is not exposed by this hook.
		return engine.Defends{Defender: p.ID, Against: against, ExtraPrevent: 3, NoExhaust: true}, msgs, true
	}})
	engine.RegisterBehavior("36013", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		var choices []engine.Choice
		for _, targetPlayer := range g.Players {
			var msgs []engine.Message
			villains := cardutil.SortedIDs(g.Villains)
			if g.ActiveVillain != "" && g.Villains[g.ActiveVillain] != nil {
				villains = []engine.EntityID{g.ActiveVillain}
			} else if len(villains) > 1 {
				villains = villains[:1]
			}
			for _, id := range villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: p.ID})
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == targetPlayer.ID {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: p.ID})
				}
			}
			msgs = append(msgs, weatherSpecial(g, p)...)
			choices = append(choices, engine.Choice{Label: engine.S(targetPlayer.Name), Kind: engine.ChoiceTarget, SourceID: targetPlayer.ID}.Msgs(msgs...))
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.blastOfWindChooseAPlayer"), choices...)}}
	}})
}

func registerStormObligation() {
	engine.RegisterBehavior("36030", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		msgs := []engine.Message{}
		if p.IsHero() {
			msgs = append(msgs, engine.ChangeForm{Player: p.ID})
		}
		// Obligations cannot remain attached to a player or lock form changes.
		// Resolve the printed alter-ego removal immediately by exhausting Ororo.
		msgs = append(msgs, engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true})
		return msgs
	}})
}

func registerStormNemesis() {
	engine.RegisterBehavior("36031", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.RevealEncounterCard)
		if !ok || m.Card.Code != "36034" {
			return nil
		}
		return []engine.Message{engine.ToughEntity{Target: e.EID()}}
	}})
	engine.RegisterBehavior("36032", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.SchemeDefeated)
		if !ok || m.Scheme != e.EID() {
			return nil
		}
		pid := cardutil.FirstPlayerID(g)
		for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
			for _, c := range *zone {
				if c.Code == "36034" {
					zone.Remove(c.ID)
					if zone == &g.EncounterDeck {
						g.ShuffleEncounterDeck()
					}
					return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: c}}
				}
			}
		}
		// The set-aside encounter zone is not represented. If no copy is in
		// the deck or discard pile, reveal a fresh deterministic copy.
		return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: engine.Card{ID: g.NextCardID(), Code: "36034"}}}
	}})
	engine.RegisterBehavior("36033", &engine.Behavior{OnAttach: func(g *engine.Game, a *engine.Attachment, target engine.EntityID) []engine.Message {
		var best *engine.Minion
		bestATK := -1
		for _, id := range cardutil.SortedIDs(g.Minions) {
			mn := g.Minions[id]
			if mn == nil {
				continue
			}
			atk := 0
			if mn.EDef().Attack != nil {
				atk = *mn.EDef().Attack
			}
			if atk > bestATK {
				best, bestATK = mn, atk
			}
		}
		if best == nil {
			g.Delete(a.ID)
			return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
		}
		a.Target = best.ID
		best.Attachments = append(best.Attachments, a.ID)
		// Piercing is not represented by the combat engine.
		return nil
	}})
	engine.RegisterBehavior("36034", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		if !p.IsHero() {
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		}
		maxATK := -1
		var targets []engine.EntityID
		for _, id := range cardutil.SortedEnemyIDs(g) {
			atk := 0
			switch enemy := g.Entity(id).(type) {
			case *engine.Villain:
				atk = enemy.AttackVal
			case *engine.Minion:
				atk = enemy.AttackVal
			}
			if atk > maxATK {
				maxATK, targets = atk, []engine.EntityID{id}
			} else if atk == maxATK {
				targets = append(targets, id)
			}
		}
		var choices []engine.Choice
		for _, id := range targets {
			enemy := g.Entity(id)
			choices = append(choices, engine.Choice{
				Label: engine.S(enemy.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode(),
			}.Msgs(
				engine.DamageEntity{Target: p.ID, Damage: maxATK, Source: id},
				engine.DamageEntity{Target: id, Damage: p.AttackStat(g), Source: p.ID},
			))
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.knifeFightChooseAnEnemyWithTheHighestAtk"), choices...)}}
	}})
}
