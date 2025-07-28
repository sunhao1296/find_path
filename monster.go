package main

type Monster struct {
	HP    int16 // 2 bytes
	ATK   int8  // 1 byte
	DEF   int8  // 1 byte
	ID    int
	Money uint8
}

type GlobalMonster struct {
	Key            string
	ID             int
	Monster        *Monster
	Pos            [2]int
	ConnectedAreas []int
}

type MonsterConnection struct {
	MonsterID      int
	Monster        *Monster
	MonsterPos     [2]int
	ConnectedAreas []int
}
