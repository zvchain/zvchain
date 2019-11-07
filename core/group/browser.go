package group

type PunishmentContext struct {
	GroupPiece *PunishmentContent
	Punish     *PunishmentContent
}
type PunishmentContent struct {
	Height      uint64
	AddressList []string
}
