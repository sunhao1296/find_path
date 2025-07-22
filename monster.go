package main

type Monster struct {
	HP    int16 // 2 bytes
	ATK   int8  // 1 byte
	DEF   int8  // 1 byte
	ID    int
	Money int16
}

type GlobalMonster struct {
	Key            string
	ID             int
	Monster        *Monster
	Money          int16
	Pos            [2]int
	ConnectedAreas []int
}

type MonsterConnection struct {
	MonsterID      int
	Monster        *Monster
	MonsterPos     [2]int
	ConnectedAreas []int
}
