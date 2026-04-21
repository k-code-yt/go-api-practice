package main

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

type EventType int

const (
	BananaEvent EventType = iota
	BananaPeel  EventType = iota
	EnergyEvent EventType = iota
	RamEvent    EventType = iota
)

var TiggerableEventsTypes []EventType = []EventType{BananaEvent, EnergyEvent, RamEvent}

// var TiggerableEventsTypes []EventType = []EventType{RamEvent}

type EventManagerState int

const (
	EventManagerState_ReadyForPickUp EventManagerState = iota
	EventManagerState_TriggerEvent   EventManagerState = iota
	EventManagerState_None           EventManagerState = iota
)

type EventActionType int

const (
	EventActionType_EventCollision EventActionType = iota
	EventActionType_SpawnPickUp    EventActionType = iota
	EventActionType_Tiggered       EventActionType = iota
	EventActionType_None           EventActionType = iota
)

type EventManager struct {
	collChecker    CollisionChecker
	bananaEventImg *ebiten.Image
	bananaPeelImg  *ebiten.Image
	energyEventImg *ebiten.Image
	state          EventManagerState
	nextEvent      PickUp
	drawItems      []PickUp
	bCount         int
	spawnTick      int
	ramEventCount  int
}

func NewEventManager(
	collChecker CollisionChecker,
	bananaEventImg *ebiten.Image,
	bananaPeelImg *ebiten.Image,
	energyEventImg *ebiten.Image,
) *EventManager {
	return &EventManager{
		collChecker:    collChecker,
		bananaEventImg: bananaEventImg,
		bananaPeelImg:  bananaPeelImg,
		energyEventImg: energyEventImg,
		state:          EventManagerState_None,
		drawItems:      []PickUp{},
	}
}

func (em *EventManager) Update(players [2]*Player) {
	if em.nextEvent != nil {
		em.nextEvent.Update(players[0])
	}
	em.HandlePickUpCollision(players)
}
func (em *EventManager) HandlePickUpCollision(players [2]*Player) {
	if em.state == EventManagerState_ReadyForPickUp {
		player := findNearestPlayer(players, em.nextEvent.GetPosition())
		isColl := em.nextEvent.IsCollidingWith(player)
		if isColl {
			em.UpdateState(EventActionType_EventCollision)
			em.nextEvent.SetLeft(!player.isLeft)

			switch em.nextEvent.EventType() {
			case EnergyEvent:
				player.Energy()
			case RamEvent:
				em.nextEvent.ProgressState()
			}
		}
	}

}

// TODO -> add state machine
func (em *EventManager) UpdateState(t EventActionType) {
	switch t {
	case EventActionType_EventCollision:
		em.state = EventManagerState_TriggerEvent
	case EventActionType_SpawnPickUp:
		em.state = EventManagerState_ReadyForPickUp
	case EventActionType_Tiggered:
		em.state = EventManagerState_None
		em.nextEvent = nil
	}
}

func (em *EventManager) SpawnPickUp(bCount int, players [2]*Player) {
	if em.state != EventManagerState_None {
		return
	}
	if em.spawnTick < eventSpawnTicks && bCount == em.bCount {
		return
	}
	em.spawnTick = 0
	em.bCount = bCount
	em.UpdateState(EventActionType_SpawnPickUp)
	position := safeSpawnPosition(em.collChecker, nil)
	var et EventType
	if em.ramEventCount >= 1 {
		et = em.pickRandomEvent([]EventType{BananaEvent, EnergyEvent})
	} else {
		et = em.pickRandomEvent(nil)
	}

	switch et {
	case BananaEvent:
		em.nextEvent = NewEventItem(position, BananaEvent)
		break
	case EnergyEvent:
		em.nextEvent = NewEventItem(position, EnergyEvent)
		break
	case RamEvent:
		p := findNearestPlayer(players, position)
		em.nextEvent = NewRam(position, p)
		em.ramEventCount++
		break
	}
}

func (em *EventManager) DrawTrigger(screen *ebiten.Image) {
	if em.nextEvent != nil && em.state == EventManagerState_TriggerEvent {
		switch em.nextEvent.EventType() {
		case BananaEvent:
			for range bananaPeelCount {
				ei := NewEventItem(
					safeSpawnPosition(em.collChecker, &CollisionOpts{
						isLeft: em.nextEvent.IsLeft(),
					}),
					BananaPeel)
				em.drawItems = append(em.drawItems, ei)
			}
		case RamEvent:
			em.drawItems = append(em.drawItems, em.nextEvent)
		}
		em.UpdateState(EventActionType_Tiggered)
	}
	if len(em.drawItems) > 0 {
		for _, ei := range em.drawItems {
			ei.DrawPickUp(screen)
		}
	}

}

func (em *EventManager) pickRandomEvent(events []EventType) EventType {
	if events == nil || len(events) == 0 {
		events = TiggerableEventsTypes
	}
	idx := rand.Intn(len(events))
	return events[idx]
}
