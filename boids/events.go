package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type EventType int

const (
	BananaEvent EventType = iota
	BananaPeel  EventType = iota
	EnergyEvent EventType = iota
)

var TiggerableEventsTypes []EventType = []EventType{BananaEvent, EnergyEvent}

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
	nextEvent      *EventItem
	drawItems      []*EventItem
	bCount         int
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
		drawItems:      []*EventItem{},
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
func (em *EventManager) HandlePickUpCollision(player *Player) {
	if em.state == EventManagerState_ReadyForPickUp {
		isColl := em.nextEvent.IsCollidingWith(player)
		if isColl {
			em.UpdateState(EventActionType_EventCollision)
			em.nextEvent.isLeft = !player.isLeft

			switch em.nextEvent.eventType {
			case EnergyEvent:
				player.Energy()
			}
		}
	}

}

func (em *EventManager) SpawnPickUp(bCount int) {
	if em.state != EventManagerState_None || bCount == em.bCount {
		return
	}
	em.bCount = bCount
	em.UpdateState(EventActionType_SpawnPickUp)
	position := safeSpawnPosition(em.collChecker, nil)
	et := em.pickRandomEvent()
	switch et {
	case BananaEvent:
		em.nextEvent = NewEventItem(position, BananaEvent)
	case EnergyEvent:
		em.nextEvent = NewEventItem(position, EnergyEvent)
	}
}

func (em *EventManager) DrawTrigger(screen *ebiten.Image) {
	if em.nextEvent != nil && em.state == EventManagerState_TriggerEvent {
		switch em.nextEvent.eventType {
		case BananaEvent:
			for range bananaPeelCount {
				ei := NewEventItem(
					safeSpawnPosition(em.collChecker, &CollisionOpts{
						isLeft: em.nextEvent.isLeft,
					}),
					BananaPeel)
				em.drawItems = append(em.drawItems, ei)
			}
		}
		em.UpdateState(EventActionType_Tiggered)
	}
	if len(em.drawItems) > 0 {
		for _, ei := range em.drawItems {
			ei.DrawPickUp(screen)
		}
	}

}

func (em *EventManager) pickRandomEvent() EventType {
	idx := rand.Intn(len(TiggerableEventsTypes))
	return TiggerableEventsTypes[idx]
}

type EventItem struct {
	position       Vector2D
	w, h           float64
	scaleX, scaleY float64
	img            *ebiten.Image
	eventType      EventType
	isLeft         bool
}

func NewEventItem(position Vector2D, eventType EventType) *EventItem {
	var img *ebiten.Image
	switch eventType {
	case BananaEvent:
		img = bananaEventSheet
		break
	case BananaPeel:
		img = bananaPeelSheet
		break
	case EnergyEvent:
		img = energySheet
		break
	default:
		panic("unknown eventType")
	}

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	scaleX := bananaSize / w
	scaleY := bananaSize / h

	return &EventItem{
		position:  position,
		w:         w,
		h:         h,
		scaleX:    scaleX,
		scaleY:    scaleY,
		img:       img,
		eventType: eventType,
	}
}

func (ei *EventItem) IsCollidingWith(p *Player) bool {
	bHW, bHH, bOffY := ei.getCollisionBox()
	bx := ei.position.x
	by := ei.position.y + bOffY

	pHW, pHH, pOffY := p.getCollisionBox()
	px := p.position.x
	py := p.position.y + pOffY

	return math.Abs(px-bx) <= pHW+bHW &&
		math.Abs(py-by) <= pHH+bHH
}

func (ei *EventItem) DrawPickUp(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}

	op.GeoM.Translate(-float64(ei.w)/2, -float64(ei.h)/2)
	op.GeoM.Scale(ei.scaleX, ei.scaleY)
	op.GeoM.Translate(ei.position.x, ei.position.y)
	screen.DrawImage(ei.img, op)

	if isDebugMode {
		ei.drawCollisionBox(screen)
	}
}

func (ei *EventItem) drawCollisionBox(screen *ebiten.Image) {
	hw, hh, offsetY := ei.getCollisionBox()
	currX := ei.position.x
	currY := ei.position.y + offsetY
	x := float32(currX - hw)
	y := float32(currY - hh)

	sideColor := func(dir Direction) color.RGBA {
		return color.RGBA{R: 255, A: 255}
	}

	// Left edge
	vector.StrokeLine(screen, float32(x), float32(y), float32(x), float32(y+float32(hh)*2), strokeWidth, sideColor(DirLeft), false)
	// Right edge
	vector.StrokeLine(screen, float32(x+float32(hw)*2), float32(y), float32(x+float32(hw)*2), float32(y+float32(hh)*2), strokeWidth, sideColor(DirRight), false)
	// Top edge
	vector.StrokeLine(screen, float32(x), float32(y), float32(x+float32(hw)*2), float32(y), strokeWidth, sideColor(DirUp), false)
	// Bottom edge
	vector.StrokeLine(screen, float32(x), float32(y+float32(hh)*2), float32(x+float32(hw)*2), float32(y+float32(hh)*2), strokeWidth, sideColor(DirDown), false)
}

func (ei *EventItem) getCollisionBox() (hw, hh, offsetY float64) {
	switch ei.eventType {
	case BananaEvent:
		hw = float64(ei.w) * ei.scaleX * bananaCollisionW
		hh = float64(ei.h) * ei.scaleY * bananaCollisionH
		offsetY = hh * collisionOffsetY
	case BananaPeel:
		hw = float64(ei.w) * ei.scaleX * bananaPeelCollisionW
		hh = float64(ei.h) * ei.scaleY * bananaPeelCollisionH
		offsetY = hh * collisionOffsetY
	}
	return
}
