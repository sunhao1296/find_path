package main

type Treasure struct {
	Type  int
	Value int8 // 1 byte
}

type TreasureItem struct {
	ID         int
	Type       int
	Value      int8
	OriginalID int
}

type GlobalTreasure struct {
	ID         string
	AreaID     int
	Type       int
	Value      int8
	OriginalID int
}
