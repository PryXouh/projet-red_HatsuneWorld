
package main

import (
    "bufio"
    "encoding/json"
    "errors"
    "fmt"
    "io/fs"
    "math"
    "math/rand"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "time"
)

const (
    saveDirName = "saves"

    stagePrologue = iota
    stageArtists
    stageMacron
    stageLabel
    stageFinish

    zoneMichael = "zone_michael"
    zoneKaaris  = "zone_kaaris"
    zoneMacron  = "zone_macron"
)

type ItemType string

type EnemyType string

const (
    itemConsumable ItemType = "consumable"
    itemEquipment  ItemType = "equipment"
    itemSpecial    ItemType = "special"
    itemMaterial   ItemType = "material"
    itemBoost      ItemType = "boost"
)

const (
    enemyHater EnemyType = "hater"
    enemyCrew  EnemyType = "crew"
    enemyRival EnemyType = "rival"
    enemyBoss  EnemyType = "boss"
    enemyFarm  EnemyType = "farm"
)

// Decrit un objet disponible dans le jeu
type ItemDefinition struct {
    ID           string
    Name         string
    Description  string
    Type         ItemType
    Price        int
    EffectID     string
    BetPointCost int
}

// Recette permettant de fabriquer un objet
type RecipeDefinition struct {
    ID        string
    Name      string
    Inputs    []string
    OutputID  string
    CraftCost int
}

// Statistiques et etat d'un personnage jouable
type Character struct {
    Name         string
    Class        string
    MaxHP        int
    HP           int
    MaxMana      int
    Mana         int
    Level        int
    XP           int
    BetPts       int
    Inventory    []string
    InventoryMax int
    Unlocked     bool
    HasNoteSpell bool
    SpecialUsed  bool

    BattleBoost int
    IgnoreGuard bool
    DodgeNext   bool
    ShieldHP    int
}

// Caracteristiques d'un adversaire
type Enemy struct {
    Name      string
    Type      EnemyType
    MaxHP     int
    HP        int
    Attack    int
    CritTimer int
    Style     string

    PoisonTurns int
    PoisonDmg   int
    WeakenTurns int
    SilenceTurns int
}

// Options qui configurent un combat
type battleOptions struct {
    AllowBet     bool
    AllowEscape  bool
    Intro        []string
    Victory      []string
    Defeat       []string
    RewardXP     int
    RewardGold   int
    RewardBetPts int
    IsBoss       bool
}

// Suit le deblocage et l'avancement d'une zone
type ZoneStatus struct {
    Unlocked  bool
    Completed bool
}

func zoneLabel(z ZoneStatus) string {
    switch {
    case z.Completed:
        return "termine"
    case z.Unlocked:
        return "accessible"
    default:
        return "verrouille"
    }
}

// Contenu serialise d'une sauvegarde
type SaveState struct {
    ProfileName     string
    PlayerIndex     int
    Characters      []Character
    StoryStage      int
    TrainingLevel   int
    TrainingBaseHP  int
    TrainingBaseAtk int
    FarmLevel       int
    CraftUnlocked   bool
    Gold            int
    Flags           map[string]bool
    ZoneStatus      map[string]ZoneStatus
    Timestamp       time.Time
}

// Gestionnaire des fichiers de sauvegarde
type SaveManager struct {
    base string
}

func newSaveManager(base string) *SaveManager {
    if base == "" {
        base = saveDirName
    }
    return &SaveManager{base: base}
}

func (sm *SaveManager) dir() string {
    if sm == nil || sm.base == "" {
        return saveDirName
    }
    return sm.base
}

func sanitizeProfileName(name string) string {
    name = strings.TrimSpace(name)
    if name == "" {
        return "profil"
    }
    var b strings.Builder
    for _, r := range name {
        switch {
        case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
            b.WriteRune(r)
        case r >= 'A' && r <= 'Z':
            b.WriteRune(r + ('a' - 'A'))
        default:
            b.WriteRune('_')
        }
    }
    out := strings.Trim(b.String(), "_")
    if out == "" {
        out = fmt.Sprintf("profil_%d", time.Now().Unix())
    }
    return out
}

func (sm *SaveManager) filePath(name string) string {
    safe := sanitizeProfileName(name)
    return filepath.Join(sm.dir(), safe+".json")
}

func (sm *SaveManager) ensureDir() error {
    return os.MkdirAll(sm.dir(), 0o755)
}

func (sm *SaveManager) save(state SaveState) error {
    if err := sm.ensureDir(); err != nil {
        return err
    }
    state.Timestamp = time.Now()
    path := sm.filePath(state.ProfileName)
    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer file.Close()
    enc := json.NewEncoder(file)
    enc.SetIndent("", "  ")
    return enc.Encode(state)
}

func (sm *SaveManager) load(name string) (*SaveState, error) {
    if err := sm.ensureDir(); err != nil {
        return nil, err
    }
    path := sm.filePath(name)
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    var state SaveState
    if err := json.NewDecoder(file).Decode(&state); err != nil {
        return nil, err
    }
    if state.ProfileName == "" {
        state.ProfileName = name
    }
    return &state, nil
}

func (sm *SaveManager) list() ([]string, error) {
    if err := sm.ensureDir(); err != nil {
        return nil, err
    }
    entries, err := os.ReadDir(sm.dir())
    if err != nil {
        if errors.Is(err, fs.ErrNotExist) {
            return []string{}, nil
        }
        return nil, err
    }
    names := make([]string, 0, len(entries))
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        if filepath.Ext(entry.Name()) != ".json" {
            continue
        }
        path := filepath.Join(sm.dir(), entry.Name())
        file, err := os.Open(path)
        if err != nil {
            continue
        }
        var state SaveState
        if err := json.NewDecoder(file).Decode(&state); err == nil {
            display := state.ProfileName
            if display == "" {
                display = strings.TrimSuffix(entry.Name(), ".json")
            }
            names = append(names, display)
        }
        file.Close()
    }
    sort.Strings(names)
    return names, nil
}

// Etat global de la partie en cours
type Game struct {
    PlayerIndex     int
    Characters      []*Character
    StoryStage      int
    TrainingLevel   int
    TrainingBaseHP  int
    TrainingBaseAtk int
    FarmLevel       int
    CraftUnlocked   bool
    Gold            int
    Flags           map[string]bool
    ZoneStatus      map[string]ZoneStatus
    rng             *rand.Rand
    saver           *SaveManager
    profile         string

    merchantItems []string
    materialItems []string
    boostItems    []string
    recipes       []RecipeDefinition

    menuReturnRequested bool
}

var activeGame *Game

func setActiveGame(g *Game) {
    activeGame = g
}

func (g *Game) consumeMenuReturn() bool {
    if g == nil || !g.menuReturnRequested {
        return false
    }
    g.menuReturnRequested = false
    return true
}

const (
    effHeal        = "heal"
    effMana        = "mana"
    effPoison      = "poison"
    effNote        = "note"
    effBag         = "bag"
    effHat         = "hat"
    effBoot        = "boot"
    effTunic       = "tunic"
    effGlove       = "glove"
    effDiscHater   = "disc_hater"
    effDiscCrew    = "disc_crew"
    effDiscBoss    = "disc_boss"
    effDiscPoison  = "disc_poison"
    effBoostX2     = "boost_x2"
    effBoostX4     = "boost_x4"
    effPass        = "pass"
    effCrew        = "crew"
)

// Catalogue des objets achetables ou trouvables
var items = map[string]ItemDefinition{
    "potion_hp":     {ID: "potion_hp", Name: "Potion de vie", Description: "Rend 50 HP", Type: itemConsumable, Price: 3, EffectID: effHeal},
    "potion_mana":   {ID: "potion_mana", Name: "Potion d'energie", Description: "Rend 20 MP", Type: itemConsumable, Price: 5, EffectID: effMana},
    "potion_poison": {ID: "potion_poison", Name: "Potion contaminee", Description: "Necessaire pour fabriquer des disques toxiques", Type: itemConsumable, Price: 6, EffectID: effPoison},
    "grimoire_note": {ID: "grimoire_note", Name: "Livre Note explosive", Description: "Apprend la note explosive", Type: itemSpecial, Price: 25, EffectID: effNote},
    "bag_upgrade":   {ID: "bag_upgrade", Name: "Extension sacoche", Description: "Ajoute 10 emplacements (max 3)", Type: itemSpecial, Price: 30, EffectID: effBag},
    "mat_loup":      {ID: "mat_loup", Name: "Sample de Loup", Description: "Sample brut", Type: itemMaterial, Price: 4},
    "mat_troll":     {ID: "mat_troll", Name: "Partition de Troll", Description: "Partition dechiree", Type: itemMaterial, Price: 7},
    "mat_sanglier":  {ID: "mat_sanglier", Name: "Cable de Sanglier", Description: "Cable sauvage", Type: itemMaterial, Price: 3},
    "mat_corb":      {ID: "mat_corb", Name: "Plume de Corbeau", Description: "Plume sombre", Type: itemMaterial, Price: 1},
    "equip_hat":     {ID: "equip_hat", Name: "Chapeau de scene", Description: "+10 HP max", Type: itemEquipment, EffectID: effHat},
    "equip_boot":    {ID: "equip_boot", Name: "Bottes de scene", Description: "+15 HP max", Type: itemEquipment, EffectID: effBoot},
    "equip_tunic":   {ID: "equip_tunic", Name: "Tunique de scene", Description: "+25 HP max", Type: itemEquipment, EffectID: effTunic},
    "equip_glove":   {ID: "equip_glove", Name: "Gant legendaire", Description: "+25 HP max", Type: itemEquipment, EffectID: effGlove},
    "disc_loup":     {ID: "disc_loup", Name: "Disque Loup", Description: "Bonus contre les haters", Type: itemSpecial, EffectID: effDiscHater},
    "disc_troll":    {ID: "disc_troll", Name: "Disque Troll", Description: "Bonus contre les crews solides", Type: itemSpecial, EffectID: effDiscCrew},
    "disc_sanglier": {ID: "disc_sanglier", Name: "Disque Sanglier", Description: "Ignore la garde des boss", Type: itemSpecial, EffectID: effDiscBoss},
    "disc_corb":     {ID: "disc_corb", Name: "Disque Corbeau", Description: "Empoisonne pendant deux tours", Type: itemSpecial, EffectID: effDiscPoison},
    "boost_x2":      {ID: "boost_x2", Name: "Boost degats x2", Description: "Double les degats pour ce combat", Type: itemBoost, BetPointCost: 15, EffectID: effBoostX2},
    "boost_x4":      {ID: "boost_x4", Name: "Boost degats x4", Description: "Degats x4 pour ce combat", Type: itemBoost, BetPointCost: 40, EffectID: effBoostX4},
    "pass_label":    {ID: "pass_label", Name: "Pass presidentiel", Description: "Ouvre l'acces au QG du label", Type: itemSpecial, EffectID: effPass},
    "crew_totem":    {ID: "crew_totem", Name: "Pouvoir d'invocation", Description: "Invoque le crew de Kaaris", Type: itemSpecial, EffectID: effCrew},
}

// Recettes disponibles chez le forgeron
var recipes = []RecipeDefinition{
    {ID: "rec_hat", Name: "Chapeau de scene", Inputs: []string{"mat_corb", "mat_sanglier"}, OutputID: "equip_hat", CraftCost: 5},
    {ID: "rec_boot", Name: "Bottes de scene", Inputs: []string{"mat_loup", "mat_sanglier"}, OutputID: "equip_boot", CraftCost: 5},
    {ID: "rec_tunic", Name: "Tunique de scene", Inputs: []string{"mat_loup", "mat_loup", "mat_troll"}, OutputID: "equip_tunic", CraftCost: 8},
    {ID: "rec_disc_l", Name: "Disque Loup", Inputs: []string{"mat_loup", "potion_poison"}, OutputID: "disc_loup", CraftCost: 0},
    {ID: "rec_disc_t", Name: "Disque Troll", Inputs: []string{"mat_troll", "potion_poison"}, OutputID: "disc_troll", CraftCost: 0},
    {ID: "rec_disc_s", Name: "Disque Sanglier", Inputs: []string{"mat_sanglier", "potion_poison"}, OutputID: "disc_sanglier", CraftCost: 0},
    {ID: "rec_disc_c", Name: "Disque Corbeau", Inputs: []string{"mat_corb", "potion_poison"}, OutputID: "disc_corb", CraftCost: 0},
}

// Implementation des effets declenches par chaque objet
var effects = map[string]func(g *Game, c *Character, enemy *Enemy) bool{
    effHeal: func(g *Game, c *Character, enemy *Enemy) bool {
        heal := 50
        if c.HP+heal > c.MaxHP {
            c.HP = c.MaxHP
        } else {
            c.HP += heal
        }
        fmt.Printf("%s boit une potion de vie (+50 HP).\n", c.Name)
        return true
    },
    effMana: func(g *Game, c *Character, enemy *Enemy) bool {
        gain := 20
        if c.Mana+gain > c.MaxMana {
            c.Mana = c.MaxMana
        } else {
            c.Mana += gain
        }
        fmt.Printf("%s retrouve 20 MP.\n", c.Name)
        return true
    },
    effPoison: func(g *Game, c *Character, enemy *Enemy) bool {
        loss := 30
        c.HP -= loss
        if c.HP < 0 {
            c.HP = 0
        }
        fmt.Println("Cette potion est trop toxique pour etre bu. Gardez-la pour le craft.")
        return true
    },
    effNote: func(g *Game, c *Character, enemy *Enemy) bool {
        if c.HasNoteSpell {
            fmt.Println("Vous connaissez deja Note explosive.")
            return false
        }
        c.HasNoteSpell = true
        fmt.Println("Note explosive apprise !")
        return true
    },
    effBag: func(g *Game, c *Character, enemy *Enemy) bool {
        if c.InventoryMax >= 40 {
            fmt.Println("Votre sacoche est deja optimisee.")
            return false
        }
        c.InventoryMax += 10
        fmt.Printf("Capacite de sacoche portee a %d objets.\n", c.InventoryMax)
        return true
    },
    effHat: func(g *Game, c *Character, enemy *Enemy) bool {
        c.MaxHP += 10
        c.HP += 10
        fmt.Println("Vous portez le Chapeau de scene : +10 HP max.")
        return true
    },
    effBoot: func(g *Game, c *Character, enemy *Enemy) bool {
        c.MaxHP += 15
        c.HP += 15
        fmt.Println("Bottes de scene equipees : +15 HP max.")
        return true
    },
    effTunic: func(g *Game, c *Character, enemy *Enemy) bool {
        c.MaxHP += 25
        c.HP += 25
        fmt.Println("Tunique de scene equipee : +25 HP max.")
        return true
    },
    effGlove: func(g *Game, c *Character, enemy *Enemy) bool {
        c.MaxHP += 25
        c.HP += 25
        fmt.Println("Le Gant legendaire pulse. +25 HP max.")
        return true
    },
    effDiscHater: func(g *Game, c *Character, enemy *Enemy) bool {
        if enemy == nil {
            fmt.Println("Ce disque doit etre utilise en combat.")
            return false
        }
        dmg := 10
        if enemy.Type == enemyHater {
            dmg += 10
        }
        enemy.HP -= dmg
        if enemy.HP < 0 {
            enemy.HP = 0
        }
        fmt.Printf("Disque de Loup : %s subit %d degats.\n", enemy.Name, dmg)
        return true
    },
    effDiscCrew: func(g *Game, c *Character, enemy *Enemy) bool {
        if enemy == nil {
            fmt.Println("Ce disque doit etre utilise en combat.")
            return false
        }
        dmg := 15
        if enemy.Type == enemyCrew {
            dmg += 15
        }
        enemy.HP -= dmg
        if enemy.HP < 0 {
            enemy.HP = 0
        }
        fmt.Printf("Disque de Troll : %s subit %d degats.\n", enemy.Name, dmg)
        return true
    },
    effDiscBoss: func(g *Game, c *Character, enemy *Enemy) bool {
        c.IgnoreGuard = true
        fmt.Println("Disque de Sanglier : votre prochaine attaque ignore la garde !")
        return true
    },
    effDiscPoison: func(g *Game, c *Character, enemy *Enemy) bool {
        if enemy == nil {
            fmt.Println("Ce disque doit etre utilise en combat.")
            return false
        }
        enemy.PoisonTurns = 2
        enemy.PoisonDmg = 5
        fmt.Printf("Disque de Corbeau : %s est empoisonne.\n", enemy.Name)
        return true
    },
    effBoostX2: func(g *Game, c *Character, enemy *Enemy) bool {
        c.BattleBoost = 2
        fmt.Printf("%s entre en mode boost : degats x2.\n", c.Name)
        return true
    },
    effBoostX4: func(g *Game, c *Character, enemy *Enemy) bool {
        c.BattleBoost = 4
        fmt.Printf("%s declenche la transe : degats x4 !\n", c.Name)
        return true
    },
    effPass: func(g *Game, c *Character, enemy *Enemy) bool {
        fmt.Println("Le pass presidentiel ouvrira certaines portes scenario.")
        return false
    },
    effCrew: func(g *Game, c *Character, enemy *Enemy) bool {
        if enemy == nil {
            fmt.Println("Personne a viser.")
            return false
        }
        dmg := 25
        enemy.HP -= dmg
        if enemy.HP < 0 {
            enemy.HP = 0
        }
        fmt.Printf("Le crew de Kaaris surgit et inflige %d degats a %s !\n", dmg, enemy.Name)
        return true
    },
}
func read(reader *bufio.Reader) string {
    line, err := reader.ReadString('\n')
    trimmed := strings.TrimSpace(line)
    if activeGame != nil && strings.EqualFold(trimmed, "menu") {
        activeGame.menuReturnRequested = true
    }
    if err != nil {
        return trimmed
    }
    return trimmed
}

func banner(title string) {
    fmt.Println()
    border := strings.Repeat("=", len(title)+8)
    fmt.Println(border)
    fmt.Printf("==  %s  ==\n", title)
    fmt.Println(border)
}

func block(reader *bufio.Reader, lines ...string) {
    for _, line := range lines {
        fmt.Println(line)
    }
    fmt.Print("[Entrer pour continuer]")
    read(reader)
    fmt.Println()
}

func shortRest(party []*Character) {
    for _, ch := range party {
        ch.ShieldHP = 0
        if ch.HP <= 0 {
            continue
        }
        ch.HP += 10
        if ch.HP > ch.MaxHP {
            ch.HP = ch.MaxHP
        }
        ch.Mana += 5
        if ch.Mana > ch.MaxMana {
            ch.Mana = ch.MaxMana
        }
    }
}
func absorbShieldDamage(target *Character, dmg int) int {
    if target == nil || target.ShieldHP <= 0 || dmg <= 0 {
        return dmg
    }
    absorbed := dmg
    if absorbed > target.ShieldHP {
        absorbed = target.ShieldHP
    }
    target.ShieldHP -= absorbed
    fmt.Printf("Le bouclier de %s absorbe %d degats.\n", target.Name, absorbed)
    return dmg - absorbed
}


func showSoloHud(player *Character, enemy *Enemy) {
    fmt.Println()
    status := fmt.Sprintf("%s | HP %d/%d | MP %d/%d | Points de mise %d", player.Name, player.HP, player.MaxHP, player.Mana, player.MaxMana, player.BetPts)
    if player.ShieldHP > 0 {
        status += fmt.Sprintf(" | Bouclier %d", player.ShieldHP)
    }
    fmt.Println(status)
    fmt.Printf("%s | HP %d/%d | ATK %d | Style %s\n\n", enemy.Name, enemy.HP, enemy.MaxHP, enemy.Attack, enemy.Style)
}



func showPartyHud(party []*Character, enemies []Enemy) {
    fmt.Println("\n-- Equipe --")
    for _, ch := range party {
        status := "KO"
        if ch.HP > 0 {
            status = fmt.Sprintf("HP %d/%d | MP %d/%d", ch.HP, ch.MaxHP, ch.Mana, ch.MaxMana)
            if ch.ShieldHP > 0 {
                status += fmt.Sprintf(" | Bouclier %d", ch.ShieldHP)
            }
        }
        fmt.Printf("%s: %s\n", ch.Name, status)
    }
    fmt.Println("-- Ennemis --")
    for i, enemy := range enemies {
        status := fmt.Sprintf("HP %d/%d", enemy.HP, enemy.MaxHP)
        if enemy.HP <= 0 {
            status = "KO"
        }
        fmt.Printf("%d) %s [%s] %s\n", i+1, enemy.Name, enemy.Style, status)
    }
    fmt.Println()
}




// Applique un objet sur un personnage et une cible eventuelle
func applyItem(g *Game, c *Character, enemy *Enemy, id string) bool {
    def, ok := items[id]
    if !ok {
        fmt.Println("Objet inconnu.")
        return false
    }
    handler := effects[def.EffectID]
    if handler == nil {
        fmt.Println("L'objet ne peut pas etre utilise ici.")
        return false
    }
    consumed := handler(g, c, enemy)
    return consumed
}

// Tente d'ajouter un objet a l'inventaire
func (c *Character) addItem(id string) bool {
    if len(c.Inventory) >= c.InventoryMax {
        fmt.Println("Votre sacoche est pleine.")
        return false
    }
    c.Inventory = append(c.Inventory, id)
    return true
}

// Retire une liste d'objets si tous sont disponibles
func (c *Character) removeItems(ids []string) bool {
    needed := map[string]int{}
    for _, id := range ids {
        needed[id]++
    }
    indexes := []int{}
    for i, have := range c.Inventory {
        if needed[have] > 0 {
            needed[have]--
            indexes = append(indexes, i)
        }
    }
    for _, rest := range needed {
        if rest > 0 {
            return false
        }
    }
    for i := len(indexes) - 1; i >= 0; i-- {
        idx := indexes[i]
        c.Inventory = append(c.Inventory[:idx], c.Inventory[idx+1:]...)
    }
    return true
}

// Ajoute de l'experience et gere les montees de niveau
func (c *Character) gainXP(amount int) {
    c.XP += amount
    for c.XP >= 100 {
        c.XP -= 100
        c.Level++
        c.MaxHP += 6
        c.MaxMana += 4
        c.HP = c.MaxHP
        c.Mana = c.MaxMana
        fmt.Printf("%s passe niveau %d !\n", c.Name, c.Level)
    }
}

// Reanime un personnage a moitie de sa vie si necessaire
func (c *Character) reviveIfNeeded() {
    if c.HP <= 0 {
        heal := c.MaxHP / 2
        if heal < 1 {
            heal = 1
        }
        c.HP = heal
        c.ShieldHP = 0
        fmt.Printf("Les fans relevent %s (%d HP).\n", c.Name, c.HP)
    }
}

// Reinitialise les etats temporaires d'un combat
func (c *Character) resetCombatFlags() {
    c.SpecialUsed = false
    c.BattleBoost = 0
    c.IgnoreGuard = false
    c.DodgeNext = false
    c.ShieldHP = 0
}

// Affiche les caracteristiques du personnage actif
func (c *Character) printStats() {
    fmt.Printf("\n%s [%s] - Niveau %d\n", c.Name, c.Class, c.Level)
    fmt.Printf("HP: %d/%d | Mana: %d/%d | XP: %d/100\n", c.HP, c.MaxHP, c.Mana, c.MaxMana, c.XP)
    fmt.Printf("Points de mise: %d | Inventaire: %d/%d\n", c.BetPts, len(c.Inventory), c.InventoryMax)
    if c.ShieldHP > 0 {
        fmt.Printf("Bouclier actif: %d HP absorbables\n", c.ShieldHP)
    }
    if c.HasNoteSpell {
        fmt.Println("Sort appris: Note explosive")
    } else {
        fmt.Println("Sort appris: aucun")
    }
}

// Construit une nouvelle partie ou recharge une sauvegarde
func newGame(sm *SaveManager, profile string, state *SaveState) *Game {
    g := &Game{
        rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
        saver:          sm,
        profile:        profile,
        merchantItems: []string{"potion_hp", "potion_mana", "potion_poison", "grimoire_note", "bag_upgrade"},
        materialItems: []string{"mat_loup", "mat_troll", "mat_sanglier", "mat_corb"},
        boostItems:    []string{"boost_x2", "boost_x4"},
        recipes:       recipes,
    }
    if state == nil {
        g.Characters = []*Character{
            {Name: "Hatsune Miku", Class: "Digital Idol", MaxHP: 80, HP: 80, MaxMana: 40, Mana: 40, Level: 1, BetPts: 30, Inventory: []string{"potion_hp", "potion_hp", "potion_hp"}, InventoryMax: 12, Unlocked: true},
            {Name: "Kaaris", Class: "Force de la Rue", MaxHP: 120, HP: 120, MaxMana: 30, Mana: 30, Level: 1, InventoryMax: 12, Unlocked: false},
            {Name: "Emmanuel Macron", Class: "Strategie Presidentielle", MaxHP: 100, HP: 100, MaxMana: 35, Mana: 35, Level: 1, InventoryMax: 12, Unlocked: false},
            {Name: "Michael Jackson", Class: "Roi de la Pop", MaxHP: 100, HP: 100, MaxMana: 35, Mana: 35, Level: 1, InventoryMax: 12, Unlocked: false},
        }
        g.ZoneStatus = map[string]ZoneStatus{
            zoneMichael: {Unlocked: true},
            zoneKaaris:  {Unlocked: true},
            zoneMacron:  {Unlocked: false},
        }
        g.Flags = map[string]bool{}
        g.TrainingBaseHP = 24
        g.TrainingBaseAtk = 5
        g.Gold = 15
        g.StoryStage = stagePrologue
        return g
    }
    g.PlayerIndex = state.PlayerIndex
    g.Characters = make([]*Character, len(state.Characters))
    for i := range state.Characters {
        ch := state.Characters[i]
        ch.resetCombatFlags()
        g.Characters[i] = &ch
    }
    g.StoryStage = state.StoryStage
    g.TrainingLevel = state.TrainingLevel
    if state.TrainingBaseHP > 0 {
        g.TrainingBaseHP = state.TrainingBaseHP
    } else {
        g.TrainingBaseHP = 24
    }
    if state.TrainingBaseAtk > 0 {
        g.TrainingBaseAtk = state.TrainingBaseAtk
    } else {
        g.TrainingBaseAtk = 5
    }
    g.FarmLevel = state.FarmLevel
    g.CraftUnlocked = state.CraftUnlocked
    g.Gold = state.Gold
    g.Flags = state.Flags
    if g.Flags == nil {
        g.Flags = map[string]bool{}
    }
    g.ZoneStatus = state.ZoneStatus
    if g.ZoneStatus == nil {
        g.ZoneStatus = map[string]ZoneStatus{}
    }
    if _, ok := g.ZoneStatus[zoneMichael]; !ok {
        g.ZoneStatus[zoneMichael] = ZoneStatus{Unlocked: true}
    }
    if _, ok := g.ZoneStatus[zoneKaaris]; !ok {
        g.ZoneStatus[zoneKaaris] = ZoneStatus{Unlocked: true}
    }
    if _, ok := g.ZoneStatus[zoneMacron]; !ok {
        g.ZoneStatus[zoneMacron] = ZoneStatus{Unlocked: false}
    }
    return g
}

// Prepare un instantane pour la sauvegarde
func (g *Game) snapshot() SaveState {
    chars := make([]Character, len(g.Characters))
    for i, ch := range g.Characters {
        copy := *ch
        copy.resetCombatFlags()
        chars[i] = copy
    }
    return SaveState{
        ProfileName:     g.profile,
        PlayerIndex:     g.PlayerIndex,
        Characters:      chars,
        StoryStage:      g.StoryStage,
        TrainingLevel:   g.TrainingLevel,
        TrainingBaseHP:  g.TrainingBaseHP,
        TrainingBaseAtk: g.TrainingBaseAtk,
        FarmLevel:       g.FarmLevel,
        CraftUnlocked:   g.CraftUnlocked,
        Gold:            g.Gold,
        Flags:           g.Flags,
        ZoneStatus:      g.ZoneStatus,
    }
}

// Sauvegarde automatiquement la progression
func (g *Game) autoSave() {
    if g.saver == nil {
        return
    }
    if err := g.saver.save(g.snapshot()); err != nil {
        fmt.Println("[Warn] sauvegarde impossible:", err)
    } else {
        fmt.Println("(Progression sauvegardee)")
    }
}

// Recupere le personnage actuellement controle
func (g *Game) active() *Character {
    if g.PlayerIndex < 0 || g.PlayerIndex >= len(g.Characters) {
        g.PlayerIndex = 0
    }
    return g.Characters[g.PlayerIndex]
}

// Donne un materiau aleatoire en recompense
func (g *Game) rewardMaterial(target *Character) {
    pool := append([]string{}, g.materialItems...)
    if len(pool) == 0 {
        return
    }
    id := pool[g.rng.Intn(len(pool))]
    if target.addItem(id) {
        fmt.Printf("Vous obtenez %s.\n", items[id].Name)
    }
}

// Interface d'utilisation des objets en combat
func (g *Game) useInventory(reader *bufio.Reader, user *Character, soloEnemy *Enemy, group []Enemy) bool {
    if len(user.Inventory) == 0 {
        fmt.Println("Votre sacoche est vide.")
        return false
    }
    fmt.Println("\n=== Inventaire ===")
    for i, id := range user.Inventory {
        if def, ok := items[id]; ok {
            fmt.Printf("%d) %s - %s\n", i+1, def.Name, def.Description)
        } else {
            fmt.Printf("%d) %s\n", i+1, id)
        }
    }
    fmt.Println("0) Retour")
    fmt.Print("Choix: ")
    choice, err := strconv.Atoi(read(reader))
    if g.consumeMenuReturn() {
        return false
    }
    if err != nil || choice < 0 || choice > len(user.Inventory) {
        fmt.Println("Choix invalide.")
        return false
    }
    if choice == 0 {
        return false
    }
    idx := choice - 1
    id := user.Inventory[idx]
    def := items[id]
    requiresTarget := false
    switch def.EffectID {
    case effDiscHater, effDiscCrew, effDiscPoison, effCrew:
        requiresTarget = true
    }
    var target *Enemy
    if requiresTarget {
        if soloEnemy != nil {
            target = soloEnemy
        } else {
            alive := 0
            for i := range group {
                if group[i].HP > 0 {
                    alive++
                }
            }
            if alive == 0 {
                fmt.Println("Aucun adversaire valide pour cet objet.")
                return false
            }
            tgt, abort := selectEnemy(reader, group)
            if abort {
                return false
            }
            if tgt == nil {
                fmt.Println("Cible invalide.")
                return false
            }
            target = tgt
        }
    } else {
        target = soloEnemy
    }
    if !applyItem(g, user, target, id) {
        return false
    }
    user.Inventory = append(user.Inventory[:idx], user.Inventory[idx+1:]...)
    return true
}


// Menu d'achat chez le disquaire
func (g *Game) handleMerchant(reader *bufio.Reader) {
    fmt.Println("\n=== Disquaire independant ===")
    listing := append([]string{}, g.merchantItems...)
    listing = append(listing, g.materialItems...)
    listing = append(listing, g.boostItems...)
    active := g.active()
    fmt.Printf("Or: %d | Points de mise: %d\n", g.Gold, active.BetPts)
    for i, id := range listing {
        def := items[id]
        price := ""
        if def.Price > 0 {
            price = fmt.Sprintf("%d or", def.Price)
        }
        if def.BetPointCost > 0 {
            if price != "" {
                price += " + "
            }
            price += fmt.Sprintf("%d pts mise", def.BetPointCost)
        }
        if price == "" {
            price = "gratuit"
        }
        fmt.Printf("%d) %s - %s (%s)\n", i+1, def.Name, def.Description, price)
    }
    fmt.Println("0) Retour")
    fmt.Print("Choix: ")
    choice, err := strconv.Atoi(read(reader))
    if g.consumeMenuReturn() {
        return
    }
    if err != nil || choice <= 0 || choice > len(listing) {
        fmt.Println("Pas d'achat.")
        return
    }
    id := listing[choice-1]
    def := items[id]
    if def.Price > 0 && g.Gold < def.Price {
        fmt.Println("Vous n'avez pas assez de fans (or).")
        return
    }
    if def.BetPointCost > 0 && active.BetPts < def.BetPointCost {
        fmt.Println("Points de mise insuffisants.")
        return
    }
    if !active.addItem(id) {
        return
    }
    g.Gold -= def.Price
    active.BetPts -= def.BetPointCost
    if active.BetPts < 0 {
        active.BetPts = 0
    }
    fmt.Printf("Vous achetez %s.\n", def.Name)
}

// Convertit les identifiants d'ingredients en noms affichables
func recipeInputs(ids []string) []string {
    out := make([]string, len(ids))
    for i, id := range ids {
        if def, ok := items[id]; ok {
            out[i] = def.Name
        } else {
            out[i] = id
        }
    }
    return out
}

// Menu de craft et de fabrication
func (g *Game) handleCraft(reader *bufio.Reader) {
    if !g.CraftUnlocked {
        fmt.Println("Le forgeron Spartan n'est pas disponible pour l'instant.")
        return
    }
    fmt.Println("\n=== Atelier Spartan ===")
    active := g.active()
    fmt.Printf("Or: %d\n", g.Gold)
    for i, rec := range g.recipes {
        fmt.Printf("%d) %s - besoin: %s | cout %d\n", i+1, rec.Name, strings.Join(recipeInputs(rec.Inputs), ", "), rec.CraftCost)
    }
    fmt.Println("0) Retour")
    fmt.Print("Choix: ")
    choice, err := strconv.Atoi(read(reader))
    if g.consumeMenuReturn() {
        return
    }
    if err != nil || choice <= 0 || choice > len(g.recipes) {
        fmt.Println("Aucun craft.")
        return
    }
    rec := g.recipes[choice-1]
    if g.Gold < rec.CraftCost {
        fmt.Println("Or insuffisant.")
        return
    }
    if !active.removeItems(rec.Inputs) {
        fmt.Println("Il vous manque des materiaux.")
        return
    }
    if !active.addItem(rec.OutputID) {
        fmt.Println("Inventaire plein, craft annule.")
        for _, id := range rec.Inputs {
            active.addItem(id)
        }
        return
    }
    g.Gold -= rec.CraftCost
    fmt.Printf("Vous forgez %s.\n", rec.Name)
}

// Pose une question a choix multiples au joueur
func (g *Game) dialogueChoice(reader *bufio.Reader, prompt string, options []string) (int, bool) {
    for {
        fmt.Println(prompt)
        for i, opt := range options {
            fmt.Printf("%d) %s\n", i+1, opt)
        }
        fmt.Print("Reponse: ")
        input := read(reader)
        if g.consumeMenuReturn() {
            return 0, true
        }
        choice, err := strconv.Atoi(input)
        if err == nil && choice >= 1 && choice <= len(options) {
            return choice - 1, false
        }
        fmt.Println("Choix invalide.")
    }
}

// Scene d'introduction et tutoriel
func (g *Game) prologue(reader *bufio.Reader) {
    banner("Chapitre 0 - Cassette volee")
    block(reader,
        "Crypton Future Media - 04h02. Les serveurs clignotent en rouge.",
        "La cassette legendaire a ete siphonnee par le label Pouler.fr.",
        "Sans elle, le chant de Miku disparaitra dans le bruit de la pub.",
    )
    fmt.Println("Manager: \"Miku, Berger et Bagland verrouillent deja tous les flux.\"")
    fmt.Println("Miku: \"Alors on va leur rappeler d'ou vient la vraie musique.\"")
    block(reader,
        "Un hater streame en direct votre chute annoncee.",
        "Montre que la scene n'appartient pas aux trolls.",
    )
    if g.consumeMenuReturn() {
        return
    }
    enemy := Enemy{Name: "Hater de studio", Type: enemyHater, MaxHP: 28, HP: 28, Attack: 4, CritTimer: 3, Style: "Troll"}
    g.fightSolo(reader, enemy, battleOptions{
        Intro:       []string{"Hater: \"Pouler.fr gere maintenant la musique legitime !\""},
        Victory:     []string{"Le live est coupe. Tes fans fideles se rassemblent."},
        RewardXP:    20,
        RewardGold:  6,
    })
    block(reader,
        "Luka: \"Quatre rivales gardent la cassette: Luka, Rin, Len et KAITO.\"",
        "Kaito: \"Cherche des allies, gagne des fans, prepare tes disques.\"",
        "Choisis tes destinations dans l'ordre que tu veux, sauf le Palais presidentiel qui attend une equipe complete.",
    )
    if g.consumeMenuReturn() {
        return
    }
    g.StoryStage = stageArtists
    g.autoSave()
}

// Hub permettant de selectionner la prochaine zone
func (g *Game) artistHub(reader *bufio.Reader) {
    for {
        banner("Carte du monde sonore")
        allies := []string{}
        for _, ch := range g.Characters {
            if ch.Name == "Hatsune Miku" {
                continue
            }
            if ch.Unlocked {
                allies = append(allies, ch.Name)
            }
        }
        if len(allies) == 0 {
            fmt.Println("Allies recrutes: aucun pour le moment.")
        } else {
            fmt.Println("Allies recrutes: " + strings.Join(allies, ", "))
        }
        fmt.Printf("Or: %d | Points de mise: %d\n", g.Gold, g.active().BetPts)
        fmt.Printf("1) Neonopolis Pop (Michael Jackson) [%s]\n", zoneLabel(g.ZoneStatus[zoneMichael]))
        fmt.Printf("2) Banlieue Rugueuse (Kaaris) [%s]\n", zoneLabel(g.ZoneStatus[zoneKaaris]))
        if g.ZoneStatus[zoneMacron].Unlocked {
            fmt.Printf("3) Palais presidentiel (Macron) [%s]\n", zoneLabel(g.ZoneStatus[zoneMacron]))
        } else {
            fmt.Println("3) Palais presidentiel (Macron) [acces refuse]")
        }
        fmt.Println("0) Retour")
        fmt.Print("Choix: ")
        choice := read(reader)
        if g.consumeMenuReturn() {
            return
        }
        switch choice {
        case "1":
            if g.ZoneStatus[zoneMichael].Completed {
                fmt.Println("MJ: \"Je suis deja avec toi. On garde le groove.\"")
            } else {
                g.zoneMichael(reader)
            }
        case "2":
            if g.ZoneStatus[zoneKaaris].Completed {
                fmt.Println("Kaaris: \"On est ensemble. File reprendre la cassette.\"")
            } else {
                g.zoneKaaris(reader)
            }
        case "3":
            if !g.ZoneStatus[zoneMacron].Unlocked {
                fmt.Println("Un agent: \"Le president attend une equipe complete, pas une idole seule.\"")
            } else if g.ZoneStatus[zoneMacron].Completed {
                fmt.Println("Macron: \"Direction le label, la Republique te regarde.\"")
            } else {
                fmt.Println("Le Palais est pret a te recevoir via l'histoire principale.")
            }
        case "0":
            return
        default:
            fmt.Println("Choix invalide.")
        }
        if g.ZoneStatus[zoneMichael].Completed && g.ZoneStatus[zoneKaaris].Completed && !g.ZoneStatus[zoneMacron].Unlocked {
            fmt.Println("Un message crypte: \"Le Palais t'ouvre ses portes.\"")
            g.ZoneStatus[zoneMacron] = ZoneStatus{Unlocked: true}
            g.StoryStage = stageMacron
            g.autoSave()
            return
        }
    }
}

// Mini-jeu de rythme pour convaincre MJ
func (g *Game) playRhythmChallenge(reader *bufio.Reader) bool {
    patterns := [][]string{
        {"MI", "KU", "MI", "KU"},
        {"POP", "ROCK", "POP"},
        {"UP", "LEFT", "RIGHT", "UP"},
        {"BEAT", "REST", "BEAT"},
        {"LA", "MI", "SO", "LA"},
    }
    seq := patterns[g.rng.Intn(len(patterns))]
    fmt.Println("MJ: \"Observe les syllabes puis renvoie-les sans erreur.\"")
    for i, syl := range seq {
        fmt.Printf("Beat %d -> %s\n", i+1, syl)
        time.Sleep(350 * time.Millisecond)
    }
    fmt.Println("Inscris la sequence sans espace (ex: POPROCKPOP):")
    raw := read(reader)
    if activeGame != nil && activeGame.consumeMenuReturn() {
        return false
    }
    input := strings.ToUpper(strings.ReplaceAll(raw, " ", ""))
    target := strings.ToUpper(strings.Join(seq, ""))
    if input == target {
        fmt.Println("MJ: \"Tu as le groove.\"")
        return true
    }
    fmt.Println("MJ: \"Tempo decale. Reviens quand tu seras calee.\"")
    return false
}

// Quete de recrutement de Michael Jackson
func (g *Game) zoneMichael(reader *bufio.Reader) {
    banner("Neonopolis Pop")
    block(reader,
        "La ville brille dans un rose synthwave.",
        "Michael Jackson glisse d'un hologramme et te fixe.",
        "MJ: \"Tu veux sauver la musique ? Montre que tu respectes le tempo.\"",
    )
    choice, abort := g.dialogueChoice(reader, "Comment repondre a MJ ?", []string{"La pop respire quand on mixe futur et nostalgie.", "Je peux t'offrir un NFT unique."})
    if abort {
        return
    }
    if choice == 1 {
        fmt.Println("MJ: \"La musique n'est pas un produit derive. Reviens quand tu ecoutes vraiment.\"")
        return
    }
    if !g.playRhythmChallenge(reader) {
        return
    }
    block(reader,
        "Les bots marketing du label saturent la place.",
        "MJ: \"On nettoie la scene.\"",
    )
    enemy := Enemy{Name: "Bot viral", Type: enemyHater, MaxHP: 60, HP: 60, Attack: 7, CritTimer: 3, Style: "Pop toxique"}
    g.fightSolo(reader, enemy, battleOptions{
        Intro:      []string{"Les bots hurlent un refrain generique."},
        Victory:    []string{"Les hologrammes repassent un clip libre."},
        RewardXP:   35,
        RewardGold: 7,
    })
    if g.consumeMenuReturn() {
        return
    }
    if !g.Characters[3].Unlocked {
        g.Characters[3].Unlocked = true
        g.Characters[3].HP = g.Characters[3].MaxHP
        g.Characters[3].Mana = g.Characters[3].MaxMana
        fmt.Println("Michael Jackson rejoint votre equipe !")
    }
    if g.active().addItem("equip_glove") {
        fmt.Println("Vous recevez le Gant legendaire.")
    }
    g.ZoneStatus[zoneMichael] = ZoneStatus{Unlocked: true, Completed: true}
    g.autoSave()
}

// Quete de recrutement de Kaaris
func (g *Game) zoneKaaris(reader *bufio.Reader) {
    banner("Banlieue Rugueuse")
    block(reader,
        "Les tours vibrent sur un kick sale.",
        "Kaaris attend, capuche en place, micro a la main.",
        "Kaaris: \"Ici on respecte le travail.\"",
    )
    if _, abort := g.dialogueChoice(reader, "Comment t'approches-tu ?", []string{"Je viens apprendre de ta scene.", "Je veux vendre des goodies."}); abort {
        return
    }
    block(reader,
        "Des haineux testent ta solidite avant le duel.",
    )
    if g.consumeMenuReturn() {
        return
    }
    g.fightSolo(reader, Enemy{Name: "Haineux de quartier", Type: enemyCrew, MaxHP: 55, HP: 55, Attack: 6, CritTimer: 3, Style: "Rue"}, battleOptions{
        AllowBet:     true,
        Intro:        []string{"Le beat tombe a 90 BPM, les coudes aussi."},
        Victory:      []string{"Le crew de reserve se retire."},
        RewardXP:     35,
        RewardGold:   6,
        RewardBetPts: 1,
    })
    if g.consumeMenuReturn() {
        return
    }
    block(reader,
        "Kaaris pose le micro entre vous.",
        "Kaaris: \"Maintenant c'est moi que tu dois convaincre.\"",
    )
    duel := Enemy{Name: "Duel avec Kaaris", Type: enemyCrew, MaxHP: 80, HP: 80, Attack: 8, CritTimer: 3, Style: "Drill"}
    if g.fightSolo(reader, duel, battleOptions{
        Intro:      []string{"Le crew entoure le ring improvise."},
        Victory:    []string{"Kaaris: \"Respect. J'entre dans ton equipe.\""},
        Defeat:     []string{"Kaaris: \"Reviens avec plus de coffre.\""},
        RewardXP:   45,
        RewardGold: 8,
    }) {
        if !g.Characters[1].Unlocked {
            g.Characters[1].Unlocked = true
            g.Characters[1].HP = g.Characters[1].MaxHP
            g.Characters[1].Mana = g.Characters[1].MaxMana
            fmt.Println("Kaaris rejoint votre equipe !")
        }
        if g.active().addItem("crew_totem") {
            fmt.Println("Vous obtenez le Pouvoir d'invocation du crew.")
        }
        if !g.CraftUnlocked {
            fmt.Println("Un ingenieur du son Spartan ouvre son atelier: le craft est desormais disponible.")
            g.CraftUnlocked = true
        }
        g.ZoneStatus[zoneKaaris] = ZoneStatus{Unlocked: true, Completed: true}
        g.autoSave()
    }
    if g.consumeMenuReturn() {
        return
    }
}

// Quete de recrutement d'Emmanuel Macron
func (g *Game) macronMission(reader *bufio.Reader) {
    status := g.ZoneStatus[zoneMacron]
    if !status.Unlocked {
        fmt.Println("Le Palais reste ferme pour le moment.")
        return
    }
    if status.Completed {
        fmt.Println("Macron: \"Cap sur le label, la Republique te soutient.\"")
        g.StoryStage = stageLabel
        return
    }
    banner("Palais presidentiel")
    block(reader,
        "Le palais ressemble a un plateau TV: marbre, drapeaux, cameras.",
        "Emmanuel Macron ajuste son micro-cravate.",
        "Macron: \"Montre-moi que tu connais la culture que tu defends.\"",
    )
    quiz := []struct{
        q string
        a string
    }{
        {"Annee du debut de la Revolution francaise ?", "1789"},
        {"Devise inscrite sur les frontons francais ?", "liberte egalite fraternite"},
        {"Compositeur de la Marseillaise ?", "rouget de lisle"},
    }
    for _, qa := range quiz {
        fmt.Println(qa.q)
        fmt.Print("Reponse: ")
        raw := read(reader)
        if g.consumeMenuReturn() {
            return
        }
        ans := strings.ToLower(strings.ReplaceAll(raw, "e", "e"))
        ans = strings.ReplaceAll(ans, "'", "")
        if ans != qa.a {
            fmt.Println("Macron: \"Reviens avec plus de fond.\"")
            return
        }
        fmt.Println("Macron hoche la tete.")
    }
    block(reader,
        "La division strategique du label tente de couper l'entretien.",
        "Macron: \"Je reste a tes cotes.\"",
    )
    g.fightSolo(reader, Enemy{Name: "Division strategique", Type: enemyCrew, MaxHP: 100, HP: 100, Attack: 11, CritTimer: 3, Style: "Lobby"}, battleOptions{
        Intro:      []string{"Les conseillers du label projectent des slides marketing."},
        Victory:    []string{"Macron brandit un badge d'acces dore."},
        RewardXP:   55,
        RewardGold: 12,
    })
    if g.consumeMenuReturn() {
        return
    }
    if !g.Characters[2].Unlocked {
        g.Characters[2].Unlocked = true
        g.Characters[2].HP = g.Characters[2].MaxHP
        g.Characters[2].Mana = g.Characters[2].MaxMana
        fmt.Println("Macron rejoint votre equipe en tant que stratege.")
    }
    if g.active().addItem("pass_label") {
        fmt.Println("Vous recevez le Pass presidentiel. Le QG peut maintenant s'ouvrir.")
    }
    g.ZoneStatus[zoneMacron] = ZoneStatus{Unlocked: true, Completed: true}
    g.StoryStage = stageLabel
    g.autoSave()
}
// Combat final contre le label Pouler.fr
func (g *Game) labelFinal(reader *bufio.Reader) {
    if !g.ZoneStatus[zoneMacron].Completed {
        fmt.Println("Rassemble Macron avant de prendre d'assaut le label.")
        return
    }
    party := g.party()
    if len(party) < 4 {
        fmt.Println("Forme d'abord ton quatuor legendaire.")
        return
    }
    banner("Chapitre 4 - Label Pouler.fr")
    block(reader,
        "Atrium du label: neon bleu, contrats encadres, foule captive.",
        "Les quatre rivales de Miku se preparent a defendre leur monopole.",
    )
    if g.consumeMenuReturn() {
        return
    }
    waveOne := []Enemy{
        {Name: "Megurine Luka", Type: enemyRival, MaxHP: 95, HP: 95, Attack: 11, CritTimer: 3, Style: "Pop aquatique"},
        {Name: "Kagamine Rin", Type: enemyRival, MaxHP: 100, HP: 100, Attack: 12, CritTimer: 3, Style: "Electro rap"},
    }
    if !g.fightParty(reader, party, waveOne, battleOptions{
        Intro:      []string{"Luka lance une ballade hypnotique, Rin tranche avec des refrains rapides."},
        Victory:    []string{"Rin: \"D'accord, Miku. Tu veux partager la scene... prouve-le.\""},
        RewardXP:   60,
        RewardGold: 15,
    }) {
        fmt.Println("Les rivales se moquent: \"Reviens avec plus de souffle.\"")
        return
    }
    shortRest(party)
    fmt.Println("La loge improvisee rend 10 HP et 5 MP a chaque allie.")
    waveTwo := []Enemy{
        {Name: "Kagamine Len", Type: enemyRival, MaxHP: 115, HP: 115, Attack: 13, CritTimer: 2, Style: "Rock urbain"},
        {Name: "KAITO", Type: enemyRival, MaxHP: 125, HP: 125, Attack: 14, CritTimer: 3, Style: "Classique glace"},
    }
    if !g.fightParty(reader, party, waveTwo, battleOptions{
        Intro:      []string{"Len sort une guitare electrique, KAITO dresse un mur symphonique."},
        Victory:    []string{"KAITO: \"La scene n'appartient a personne. Gagne ton final.\""},
        RewardXP:   70,
        RewardGold: 18,
    }) {
        fmt.Println("Len: \"On vous attend pour une vraie bagarre.\"")
        return
    }
    block(reader,
        "Mattieu Berger et Sylvain Bagland applaudissent avec arrogance.",
        "Ils declenchent des cages de verre autour de tes allies.",
        "Miku se retrouve seule au centre de la scene.",
    )
    solo := []*Character{g.Characters[0]}
    g.Characters[0].resetCombatFlags()
    bosses := []Enemy{
        {Name: "Mattieu Berger", Type: enemyBoss, MaxHP: 165, HP: 165, Attack: 15, CritTimer: 3, Style: "Business"},
        {Name: "Sylvain Bagland", Type: enemyBoss, MaxHP: 155, HP: 155, Attack: 15, CritTimer: 2, Style: "Business"},
    }
    if !g.fightParty(reader, solo, bosses, battleOptions{
        Intro: []string{"Berger: \"Sans ta cassette tu n'es rien.\"", "Bagland: \"La musique se monetise, point.\""},
        Victory: []string{"La cassette legendaire scintille de nouveau entre les mains de Miku."},
        Defeat:  []string{"Berger: \"Le marche decide. Reviens avec plus de fans.\""},
        RewardXP:   120,
        RewardGold: 25,
        IsBoss:     true,
    }) {
        fmt.Println("Les dirigeants sourient: \"On te verra a la prochaine sortie.\"")
        return
    }
    block(reader,
        "Les cages explosent, tes allies te rejoignent.",
        "Miku remet la cassette dans son lecteur: le monde entier recoit a nouveau des melodies libres.",
        "La vraie musique appartient aux artistes et au public, pas aux labels.",
    )
    if g.consumeMenuReturn() {
        return
    }
    g.StoryStage = stageFinish
    g.autoSave()
}
// Valeur d'attaque de base selon le personnage
func baseAttack(c *Character) int {
    switch c.Name {
    case "Kaaris":
        return 12
    case "Michael Jackson":
        return 10
    case "Emmanuel Macron":
        return 9
    default:
        return 9
    }
}


// Gere les capacites speciales contextuelles
func (g *Game) performSpecial(reader *bufio.Reader, c *Character, enemy *Enemy, party []*Character) (bool, bool) {
    if c == nil {
        return false, false
    }
    switch c.Name {
    case "Hatsune Miku":
        if !c.HasNoteSpell {
            fmt.Println("Miku n'a pas encore retrouve la note explosive.")
            return false, false
        }
        if enemy == nil {
            fmt.Println("Aucune cible a pulveriser.")
            return false, false
        }
        cost := 15
        if c.Mana < cost {
            fmt.Println("Pas assez de mana pour la note explosive legendaire.")
            return false, false
        }
        c.Mana -= cost
        dmg := 30 + g.rng.Intn(11)
        if c.BattleBoost > 0 {
            dmg *= c.BattleBoost
        }
        if c.IgnoreGuard {
            dmg += 8
            c.IgnoreGuard = false
        }
        enemy.HP -= dmg
        if enemy.HP < 0 {
            enemy.HP = 0
        }
        fmt.Printf("Miku declenche la note explosive legendaire (-%d HP).\n", dmg)
        c.SpecialUsed = true
        return true, true
    case "Kaaris":
        fmt.Println("Kaaris: \"On choisit quoi ?\"")
        fmt.Println("1) Crew devastateur (0 MP)")
        fmt.Println("2) Bouclier de rue (-10 MP)")
        fmt.Println("3) Mur du crew (-18 MP)")
        fmt.Print("Choix: ")
        choice := read(reader)
        if g.consumeMenuReturn() {
            return false, false
        }
        switch choice {
        case "1":
            if enemy == nil {
                fmt.Println("Pas de cible pour frapper.")
                return false, false
            }
            dmg := 34 + g.rng.Intn(13)
            if c.BattleBoost > 0 {
                dmg *= c.BattleBoost
            }
            if c.IgnoreGuard {
                dmg += 10
                c.IgnoreGuard = false
            }
            enemy.HP -= dmg
            if enemy.HP < 0 {
                enemy.HP = 0
            }
            fmt.Printf("Kaaris invoque son crew (-%d HP).\n", dmg)
            c.SpecialUsed = true
            return true, true
        case "2":
            cost := 10
            if c.Mana < cost {
                fmt.Println("Pas assez de mana pour lever le bouclier.")
                return false, false
            }
            c.Mana -= cost
            shield := 24
            c.ShieldHP += shield
            fmt.Printf("Un bouclier d'acier entoure %s (+%d HP absorbables).\n", c.Name, shield)
            c.SpecialUsed = true
            return true, true
        case "3":
            cost := 18
            if c.Mana < cost {
                fmt.Println("Pas assez de mana pour proteger tout le monde.")
                return false, false
            }
            c.Mana -= cost
            applied := 0
            for _, ally := range party {
                if ally == nil || ally.HP <= 0 {
                    continue
                }
                ally.ShieldHP += 18
                applied++
            }
            if applied == 0 {
                fmt.Println("Personne a proteger.")
                return false, false
            }
            if applied == 1 {
                fmt.Println("Le crew forme un bouclier autour de toi (+18 HP absorbables).")
            } else {
                fmt.Println("Le crew erige un mur protecteur pour l'equipe (+18 HP absorbables chacun).")
            }
            c.SpecialUsed = true
            return true, true
        default:
            fmt.Println("Choix invalide.")
            return false, false
        }
    case "Emmanuel Macron":
        if enemy == nil {
            fmt.Println("Aucune cible politique en face.")
            return false, false
        }
        fmt.Println("Macron: \"Quelle tactique ?\"")
        fmt.Println("1) Discours manipulateur (-12 MP)")
        fmt.Println("2) Interdiction de chanter (-14 MP)")
        fmt.Print("Choix: ")
        choice := read(reader)
        if g.consumeMenuReturn() {
            return false, false
        }
        switch choice {
        case "1":
            cost := 12
            if c.Mana < cost {
                fmt.Println("Pas assez d'energie pour le discours manipulateur.")
                return false, false
            }
            c.Mana -= cost
            if enemy.WeakenTurns < 2 {
                enemy.WeakenTurns = 2
            }
            fmt.Printf("Macron deboussole %s : ses degats sont divises pendant 2 tours.\n", enemy.Name)
            c.SpecialUsed = true
            return true, true
        case "2":
            cost := 14
            if c.Mana < cost {
                fmt.Println("Pas assez d'energie pour l'interdiction de chanter.")
                return false, false
            }
            c.Mana -= cost
            enemy.SilenceTurns = 1
            fmt.Printf("%s recoit une interdiction de chanter et ne pourra pas attaquer ce tour-ci.\n", enemy.Name)
            c.SpecialUsed = true
            return true, false
        default:
            fmt.Println("Choix invalide.")
            return false, false
        }
    case "Michael Jackson":
        fmt.Println("MJ: \"Choisis ton groove.\"")
        fmt.Println("1) Moonwalk offensif (-8 MP)")
        fmt.Println("2) Beat therapy (-12 MP, soin perso)")
        fmt.Println("3) Harmonie partagee (-18 MP, soigne l'equipe)")
        fmt.Print("Choix: ")
        choice := read(reader)
        if g.consumeMenuReturn() {
            return false, false
        }
        switch choice {
        case "1":
            if enemy == nil {
                fmt.Println("Le moonwalk attend un adversaire.")
                return false, false
            }
            cost := 8
            if c.Mana < cost {
                fmt.Println("Pas assez d'energie pour le moonwalk.")
                return false, false
            }
            c.Mana -= cost
            dmg := 20 + g.rng.Intn(9)
            if c.BattleBoost > 0 {
                dmg *= c.BattleBoost
            }
            if c.IgnoreGuard {
                dmg += 6
                c.IgnoreGuard = false
            }
            enemy.HP -= dmg
            if enemy.HP < 0 {
                enemy.HP = 0
            }
            c.DodgeNext = true
            fmt.Printf("MJ glisse en moonwalk et inflige %d degats. Il esquivera le prochain coup.\n", dmg)
            c.SpecialUsed = true
            return true, true
        case "2":
            cost := 12
            if c.Mana < cost {
                fmt.Println("Pas assez d'energie pour ce solo.")
                return false, false
            }
            c.Mana -= cost
            heal := 32
            c.HP += heal
            if c.HP > c.MaxHP {
                c.HP = c.MaxHP
            }
            fmt.Printf("MJ improvise un solo apaisant et se soigne (+%d HP).\n", heal)
            c.SpecialUsed = true
            return true, true
        case "3":
            cost := 18
            if c.Mana < cost {
                fmt.Println("Pas assez d'energie pour harmoniser l'equipe.")
                return false, false
            }
            c.Mana -= cost
            healed := 0
            for _, ally := range party {
                if ally == nil || ally.HP <= 0 {
                    continue
                }
                gain := 20
                ally.HP += gain
                if ally.HP > ally.MaxHP {
                    ally.HP = ally.MaxHP
                }
                healed++
            }
            if healed == 0 {
                fmt.Println("Personne n'est en etat de profiter de l'harmonie.")
                return false, false
            }
            fmt.Println("Le choeur de MJ guerit l'equipe (+20 HP chacun).")
            c.SpecialUsed = true
            return true, true
        default:
            fmt.Println("Choix invalide.")
            return false, false
        }
    default:
        fmt.Println("Pas de capacite speciale propre.")
        return false, false
    }
}




// Boucle de combat pour les duels
func (g *Game) fightSolo(reader *bufio.Reader, enemy Enemy, opts battleOptions) bool {
    player := g.active()
    player.resetCombatFlags()
    enemy.HP = enemy.MaxHP
    if enemy.CritTimer <= 0 {
        enemy.CritTimer = 3
    }
    enemy.SilenceTurns = 0
    for _, line := range opts.Intro {
        fmt.Println("[INFO]", line)
    }
    bet := 1
    if opts.AllowBet && player.BetPts > 0 {
        fmt.Printf("Points de mise disponibles: %d (0 aucun, 2/3/4 pour miser) -> ", player.BetPts)
        betInput := read(reader)
        if g.consumeMenuReturn() {
            fmt.Println("Retour au menu principal.")
            return false
        }
        switch betInput {
        case "2":
            if player.BetPts >= 2 {
                bet = 2
            }
        case "3":
            if player.BetPts >= 3 {
                bet = 3
            }
        case "4":
            if player.BetPts >= 4 {
                bet = 4
            }
        }
    }
    enemy.HP *= bet
    enemy.MaxHP = enemy.HP
    enemy.Attack = int(float64(enemy.Attack) * math.Sqrt(float64(bet)))
    turn := 1
    for enemy.HP > 0 && player.HP > 0 {
        showSoloHud(player, &enemy)
        fmt.Printf("Tour %d\n", turn)
        hasNyan := player.Name == "Hatsune Miku"
        fmt.Println("1) Attaquer")
        if player.HasNoteSpell {
            fmt.Println("2) Note explosive")
        } else {
            fmt.Println("2) Note explosive (verrouille)")
        }
        if hasNyan {
            fmt.Println("3) Attaque Nyan Cat")
            fmt.Println("4) Capacite speciale")
            fmt.Println("5) Inventaire")
            fmt.Println("6) Observer")
            if opts.AllowEscape {
                fmt.Println("7) Fuir")
            }
        } else {
            fmt.Println("3) Capacite speciale")
            fmt.Println("4) Inventaire")
            fmt.Println("5) Observer")
            if opts.AllowEscape {
                fmt.Println("6) Fuir")
            }
        }
        fmt.Print("Action: ")
        action := read(reader)
        if g.consumeMenuReturn() {
            fmt.Println("Retour au menu principal.")
            return false
        }
        consumeTurn := true
        switch action {
        case "1":
            dmg := baseAttack(player) + g.rng.Intn(4)
            if player.BattleBoost > 0 {
                dmg *= player.BattleBoost
            }
            if player.IgnoreGuard {
                dmg += 6
                player.IgnoreGuard = false
            }
            enemy.HP -= dmg
            if enemy.HP < 0 {
                enemy.HP = 0
            }
            fmt.Printf("%s inflige %d degats.\n", player.Name, dmg)
        case "2":
            if !player.HasNoteSpell {
                fmt.Println("Vous n'avez pas encore appris ce sort.")
                consumeTurn = false
            } else if player.Mana < 10 {
                fmt.Println("Pas assez de mana.")
                consumeTurn = false
            } else {
                player.Mana -= 10
                dmg := 18 + g.rng.Intn(6)
                if player.BattleBoost > 0 {
                    dmg *= player.BattleBoost
                }
                if player.IgnoreGuard {
                    dmg += 8
                    player.IgnoreGuard = false
                }
                enemy.HP -= dmg
                if enemy.HP < 0 {
                    enemy.HP = 0
                }
                fmt.Printf("Note explosive inflige %d degats.\n", dmg)
            }
        case "3":
            if hasNyan {
                manaCost := 16
                if player.Mana < manaCost {
                    fmt.Println("Pas assez de mana pour invoquer Nyan Cat.")
                    consumeTurn = false
                } else {
                    player.Mana -= manaCost
                    dmg := 26 + g.rng.Intn(8)
                    if player.BattleBoost > 0 {
                        dmg *= player.BattleBoost
                    }
                    if player.IgnoreGuard {
                        dmg += 10
                        player.IgnoreGuard = false
                    }
                    enemy.HP -= dmg
                    if enemy.HP < 0 {
                        enemy.HP = 0
                    }
                    fmt.Printf("Nyan Cat dechaine son arc-en-ciel et inflige %d degats !\n", dmg)
                }
            } else {
                if player.SpecialUsed {
                    fmt.Println("Capacite deja utilisee.")
                    consumeTurn = false
                } else {
                    used, consume := g.performSpecial(reader, player, &enemy, []*Character{player})
                    if g.consumeMenuReturn() {
                        fmt.Println("Retour au menu principal.")
                        return false
                    }
                    if !used {
                        consumeTurn = false
                    } else if !consume {
                        consumeTurn = false
                    }
                }
            }
        case "4":
            if hasNyan {
                if player.SpecialUsed {
                    fmt.Println("Capacite deja utilisee.")
                    consumeTurn = false
                } else {
                    used, consume := g.performSpecial(reader, player, &enemy, []*Character{player})
                    if g.consumeMenuReturn() {
                        fmt.Println("Retour au menu principal.")
                        return false
                    }
                    if !used {
                        consumeTurn = false
                    } else if !consume {
                        consumeTurn = false
                    }
                }
            } else {
                if !g.useInventory(reader, player, &enemy, nil) {
                    consumeTurn = false
                }
                if g.consumeMenuReturn() {
                    fmt.Println("Retour au menu principal.")
                    return false
                }
            }
        case "5":
            if hasNyan {
                if !g.useInventory(reader, player, &enemy, nil) {
                    consumeTurn = false
                }
                if g.consumeMenuReturn() {
                    fmt.Println("Retour au menu principal.")
                    return false
                }
            } else {
                fmt.Printf("%s (%s) HP %d/%d | ATK %d\n", enemy.Name, enemy.Style, enemy.HP, enemy.MaxHP, enemy.Attack)
                consumeTurn = false
            }
        case "6":
            if hasNyan {
                fmt.Printf("%s (%s) HP %d/%d | ATK %d\n", enemy.Name, enemy.Style, enemy.HP, enemy.MaxHP, enemy.Attack)
                consumeTurn = false
            } else if opts.AllowEscape {
                fmt.Println("Vous battez en retraite.")
                return false
            } else {
                fmt.Println("Impossible de fuir.")
                consumeTurn = false
            }
        case "7":
            if hasNyan {
                if opts.AllowEscape {
                    fmt.Println("Vous battez en retraite.")
                    return false
                }
                fmt.Println("Impossible de fuir.")
                consumeTurn = false
            } else {
                fmt.Println("Action inconnue.")
                consumeTurn = false
            }
        default:
            fmt.Println("Action inconnue.")
            consumeTurn = false
        }

        if enemy.HP <= 0 {
            break
        }

        if consumeTurn {
            if enemy.PoisonTurns > 0 {
                enemy.HP -= enemy.PoisonDmg
                if enemy.HP < 0 {
                    enemy.HP = 0
                }
                fmt.Printf("Le poison ronge %s (-%d HP).\n", enemy.Name, enemy.PoisonDmg)
                enemy.PoisonTurns--
                if enemy.HP <= 0 {
                    break
                }
            }
            if enemy.SilenceTurns > 0 {
                fmt.Printf("%s est reduit au silence et ne peut pas attaquer.\n", enemy.Name)
                enemy.SilenceTurns--
                if enemy.CritTimer > 1 {
                    enemy.CritTimer--
                }
                turn++
                continue
            }
            dmg := enemy.Attack
            if enemy.WeakenTurns > 0 {
                dmg = int(math.Round(float64(dmg) * 0.6))
                if dmg < 1 {
                    dmg = 1
                }
                enemy.WeakenTurns--
            }
            if enemy.CritTimer <= 1 {
                dmg *= 2
                enemy.CritTimer = 3
                fmt.Println("L'ennemi place un critique !")
            } else {
                enemy.CritTimer--
            }
            if player.DodgeNext {
                fmt.Printf("%s esquive le coup !\n", player.Name)
                player.DodgeNext = false
            } else {
                dmg = absorbShieldDamage(player, dmg)
                if dmg > 0 {
                    player.HP -= dmg
                    if player.HP < 0 {
                        player.HP = 0
                    }
                    fmt.Printf("%s subit %d degats.\n", player.Name, dmg)
                }
            }
        }
        turn++
    }
    if enemy.HP <= 0 {
        fmt.Println("Victoire !")
        xpGain := opts.RewardXP * bet
        if xpGain > 0 {
            player.gainXP(xpGain)
        }
        goldGain := opts.RewardGold * bet
        if goldGain > 0 {
            g.Gold += goldGain
        }
        if opts.AllowBet && bet > 1 {
            player.BetPts += bet - 1
            fmt.Printf("Gain de mise: +%d (total %d).\n", bet-1, player.BetPts)
        }
        if opts.RewardBetPts > 0 {
            player.BetPts += opts.RewardBetPts * bet
            fmt.Printf("Points de mise bonus: +%d.\n", opts.RewardBetPts*bet)
        }
        if goldGain > 0 || xpGain > 0 {
            fmt.Printf("Recompenses: +%d or | +%d XP\n", goldGain, xpGain)
        }
        for _, line := range opts.Victory {
            fmt.Println(line)
        }
        return true
    }
    fmt.Println("Defaite...")
    player.reviveIfNeeded()
    if opts.AllowBet && bet > 1 {
        player.BetPts -= bet
        if player.BetPts < 0 {
            player.BetPts = 0
        }
    }
    for _, line := range opts.Defeat {
        fmt.Println(line)
    }
    return false
}



// Indique si tous les ennemis sont vaincus
func allEnemiesDown(enemies []Enemy) bool {
    for _, e := range enemies {
        if e.HP > 0 {
            return false
        }
    }
    return true
}

// Verifie si toute l'equipe est KO
func allAlliesDown(party []*Character) bool {
    for _, c := range party {
        if c.HP > 0 {
            return false
        }
    }
    return true
}

// Choisit un allie vivant au hasard
func targetAlive(rng *rand.Rand, party []*Character) *Character {
    alive := []*Character{}
    for _, ch := range party {
        if ch.HP > 0 {
            alive = append(alive, ch)
        }
    }
    if len(alive) == 0 {
        return nil
    }
    return alive[rng.Intn(len(alive))]
}

// Gestion des combats de groupe
func (g *Game) fightParty(reader *bufio.Reader, party []*Character, enemies []Enemy, opts battleOptions) bool {
    for _, ch := range party {
        ch.resetCombatFlags()
        ch.reviveIfNeeded()
    }
    for i := range enemies {
        enemies[i].HP = enemies[i].MaxHP
        if enemies[i].CritTimer <= 0 {
            enemies[i].CritTimer = 3
        }
        enemies[i].SilenceTurns = 0
    }
    for _, line := range opts.Intro {
        fmt.Println("[INFO]", line)
    }
    round := 1
    for {
        if allEnemiesDown(enemies) {
            fmt.Println("Victoire du groupe !")
            if opts.RewardXP > 0 {
                for _, ch := range party {
                    ch.gainXP(opts.RewardXP)
                }
            }
            if opts.RewardGold > 0 {
                g.Gold += opts.RewardGold
            }
            if opts.RewardGold > 0 || opts.RewardXP > 0 {
                fmt.Printf("Recompenses: +%d or | +%d XP par allie\n", opts.RewardGold, opts.RewardXP)
            }
            for _, line := range opts.Victory {
                fmt.Println(line)
            }
            return true
        }
        if allAlliesDown(party) {
            fmt.Println("L'equipe tombe !")
            for _, ch := range party {
                ch.reviveIfNeeded()
            }
            for _, line := range opts.Defeat {
                fmt.Println(line)
            }
            return false
        }
        showPartyHud(party, enemies)
        fmt.Printf("Tour %d\n", round)
        for _, ch := range party {
            if ch.HP <= 0 {
                continue
            }
            for {
                fmt.Printf("\n%s (HP %d/%d | MP %d/%d", ch.Name, ch.HP, ch.MaxHP, ch.Mana, ch.MaxMana)
                if ch.ShieldHP > 0 {
                    fmt.Printf(" | Bouclier %d", ch.ShieldHP)
                }
                fmt.Println(")")
                hasNyan := ch.Name == "Hatsune Miku"
                fmt.Println("1) Attaquer")
                if ch.HasNoteSpell {
                    fmt.Println("2) Note explosive")
                } else {
                    fmt.Println("2) Note explosive (verrouille)")
                }
                if hasNyan {
                    fmt.Println("3) Attaque Nyan Cat")
                    fmt.Println("4) Capacite speciale")
                    fmt.Println("5) Inventaire")
                    fmt.Println("6) Observer")
                    if opts.AllowEscape {
                        fmt.Println("7) Fuir")
                    }
                } else {
                    fmt.Println("3) Capacite speciale")
                    fmt.Println("4) Inventaire")
                    fmt.Println("5) Observer")
                    if opts.AllowEscape {
                        fmt.Println("6) Fuir")
                    }
                }
                fmt.Print("Action: ")
                action := read(reader)
                if g.consumeMenuReturn() {
                    fmt.Println("Retour au menu principal.")
                    return false
                }
                consumeTurn := true
                handled := true

                switch action {
                case "1":
                    target, abort := selectEnemy(reader, enemies)
                    if abort {
                        fmt.Println("Retour au menu principal.")
                        return false
                    }
                    if target == nil {
                        handled = false
                        consumeTurn = false
                    } else {
                        dmg := baseAttack(ch) + g.rng.Intn(5)
                        if ch.BattleBoost > 0 {
                            dmg *= ch.BattleBoost
                        }
                        if ch.IgnoreGuard {
                            dmg += 6
                            ch.IgnoreGuard = false
                        }
                        target.HP -= dmg
                        if target.HP < 0 {
                            target.HP = 0
                        }
                        fmt.Printf("%s frappe %s pour %d degats.\n", ch.Name, target.Name, dmg)
                    }
                case "2":
                    if !ch.HasNoteSpell || ch.Mana < 10 {
                        fmt.Println("Sort indisponible.")
                        handled = false
                        consumeTurn = false
                    } else {
                        target, abort := selectEnemy(reader, enemies)
                        if abort {
                            fmt.Println("Retour au menu principal.")
                            return false
                        }
                        if target == nil {
                            handled = false
                            consumeTurn = false
                        } else {
                            ch.Mana -= 10
                            dmg := 18 + g.rng.Intn(7)
                            if ch.BattleBoost > 0 {
                                dmg *= ch.BattleBoost
                            }
                            if ch.IgnoreGuard {
                                dmg += 8
                                ch.IgnoreGuard = false
                            }
                            target.HP -= dmg
                            if target.HP < 0 {
                                target.HP = 0
                            }
                            fmt.Printf("Note explosive touche %s pour %d degats.\n", target.Name, dmg)
                        }
                    }
                case "3":
                    if hasNyan {
                        if ch.Mana < 16 {
                            fmt.Println("Pas assez de mana pour invoquer Nyan Cat.")
                            handled = false
                            consumeTurn = false
                        } else {
                            target, abort := selectEnemy(reader, enemies)
                            if abort {
                                fmt.Println("Retour au menu principal.")
                                return false
                            }
                            if target == nil {
                                handled = false
                                consumeTurn = false
                            } else {
                                ch.Mana -= 16
                                dmg := 26 + g.rng.Intn(8)
                                if ch.BattleBoost > 0 {
                                    dmg *= ch.BattleBoost
                                }
                                if ch.IgnoreGuard {
                                    dmg += 10
                                    ch.IgnoreGuard = false
                                }
                                target.HP -= dmg
                                if target.HP < 0 {
                                    target.HP = 0
                                }
                                fmt.Printf("Nyan Cat dechire la scene et inflige %d degats a %s !\n", dmg, target.Name)
                            }
                        }
                    } else {
                        if ch.SpecialUsed {
                            fmt.Println("Capacite deja utilisee.")
                            handled = false
                            consumeTurn = false
                        } else {
                            idx := firstAliveEnemy(enemies)
                            if idx == -1 {
                                handled = false
                                consumeTurn = false
                            } else {
                                used, consume := g.performSpecial(reader, ch, &enemies[idx], party)
                                if g.consumeMenuReturn() {
                                    fmt.Println("Retour au menu principal.")
                                    return false
                                }
                                if !used {
                                    handled = false
                                    consumeTurn = false
                                } else if !consume {
                                    consumeTurn = false
                                }
                            }
                        }
                    }
                case "4":
                    if hasNyan {
                        if ch.SpecialUsed {
                            fmt.Println("Capacite deja utilisee.")
                            handled = false
                            consumeTurn = false
                        } else {
                            idx := firstAliveEnemy(enemies)
                            if idx == -1 {
                                handled = false
                                consumeTurn = false
                            } else {
                                used, consume := g.performSpecial(reader, ch, &enemies[idx], party)
                                if g.consumeMenuReturn() {
                                    fmt.Println("Retour au menu principal.")
                                    return false
                                }
                                if !used {
                                    handled = false
                                    consumeTurn = false
                                } else if !consume {
                                    consumeTurn = false
                                }
                            }
                        }
                    } else {
                        if !g.useInventory(reader, ch, nil, enemies) {
                            handled = false
                            consumeTurn = false
                        }
                        if g.consumeMenuReturn() {
                            fmt.Println("Retour au menu principal.")
                            return false
                        }
                    }
                case "5":
                    if hasNyan {
                        if !g.useInventory(reader, ch, nil, enemies) {
                            handled = false
                            consumeTurn = false
                        }
                        if g.consumeMenuReturn() {
                            fmt.Println("Retour au menu principal.")
                            return false
                        }
                    } else {
                        printEnemies(enemies)
                        consumeTurn = false
                    }
                case "6":
                    if hasNyan {
                        printEnemies(enemies)
                        consumeTurn = false
                    } else if opts.AllowEscape {
                        fmt.Println("Vous battez en retraite.")
                        return false
                    } else {
                        fmt.Println("Impossible de fuir.")
                        handled = false
                        consumeTurn = false
                    }
                case "7":
                    if hasNyan {
                        if opts.AllowEscape {
                            fmt.Println("Vous battez en retraite.")
                            return false
                        }
                        fmt.Println("Impossible de fuir.")
                        handled = false
                        consumeTurn = false
                    } else {
                        fmt.Println("Action inconnue.")
                        handled = false
                        consumeTurn = false
                    }
                default:
                    fmt.Println("Action inconnue.")
                    handled = false
                    consumeTurn = false
                }

                if !handled {
                    continue
                }
                if !consumeTurn {
                    continue
                }
                break
            }
        }

        if allEnemiesDown(enemies) {
            continue
        }

        for i := range enemies {
            enemy := &enemies[i]
            if enemy.HP <= 0 {
                continue
            }
            if enemy.PoisonTurns > 0 {
                enemy.HP -= enemy.PoisonDmg
                if enemy.HP < 0 {
                    enemy.HP = 0
                }
                fmt.Printf("%s souffre du poison (-%d).\n", enemy.Name, enemy.PoisonDmg)
                enemy.PoisonTurns--
                if enemy.HP <= 0 {
                    continue
                }
            }
            if enemy.SilenceTurns > 0 {
                fmt.Printf("%s est reduit au silence et ne peut pas attaquer.\n", enemy.Name)
                enemy.SilenceTurns--
                if enemy.CritTimer > 1 {
                    enemy.CritTimer--
                }
                continue
            }
            target := targetAlive(g.rng, party)
            if target == nil {
                continue
            }
            dmg := enemy.Attack
            if enemy.WeakenTurns > 0 {
                dmg = int(math.Round(float64(dmg) * 0.6))
                if dmg < 1 {
                    dmg = 1
                }
                enemy.WeakenTurns--
            }
            if enemy.CritTimer <= 1 {
                dmg *= 2
                enemy.CritTimer = 3
                fmt.Printf("%s declenche un critique !\n", enemy.Name)
            } else {
                enemy.CritTimer--
            }
            if target.DodgeNext {
                fmt.Printf("%s esquive grace au moonwalk !\n", target.Name)
                target.DodgeNext = false
                continue
            }
            dmg = absorbShieldDamage(target, dmg)
            if dmg <= 0 {
                continue
            }
            target.HP -= dmg
            if target.HP < 0 {
                target.HP = 0
            }
            fmt.Printf("%s inflige %d degats a %s.\n", enemy.Name, dmg, target.Name)
        }
        round++
    }
}


// Renvoie l'indice du premier ennemi encore debout
func firstAliveEnemy(enemies []Enemy) int {
    for i, e := range enemies {
        if e.HP > 0 {
            return i
        }
    }
    return -1
}

// Selectionne une cible ennemie via le joueur
func selectEnemy(reader *bufio.Reader, enemies []Enemy) (*Enemy, bool) {
    if len(enemies) == 0 {
        return nil, false
    }
    for {
        fmt.Print("Cible (numero): ")
        input := read(reader)
        if activeGame != nil && activeGame.consumeMenuReturn() {
            return nil, true
        }
        idx, err := strconv.Atoi(input)
        if err != nil || idx <= 0 || idx > len(enemies) {
            fmt.Println("Cible invalide.")
            continue
        }
        if enemies[idx-1].HP <= 0 {
            fmt.Println("Cette cible est deja a terre.")
            continue
        }
        return &enemies[idx-1], false
    }
}

// Liste les allies actuellement disponibles
func (g *Game) party() []*Character {
    var out []*Character
    for _, ch := range g.Characters {
        if ch.Unlocked {
            out = append(out, ch)
        }
    }
    return out
}

// Menu pause accessible pendant un combat
func (g *Game) battlePause(reader *bufio.Reader) string {
    fmt.Println("\n=== Pause combat ===")
    fmt.Println("1) Reprendre")
    fmt.Println("2) Quitter le combat")
    fmt.Println("3) Sauvegarder")
    fmt.Println("4) Statistiques")
    fmt.Print("Choix: ")
    choice := read(reader)
    switch choice {
    case "1":
        return "resume"
    case "2":
        return "abort"
    case "3":
        g.autoSave()
    case "4":
        g.active().printStats()
    default:
        fmt.Println("Choix invalide.")
    }
    return "resume"
}

// Session d'entrainement pour ameliorer l'equipe
func (g *Game) training(reader *bufio.Reader) {
    fmt.Println("\n=== Entrainement ===")
    hp := g.TrainingBaseHP + g.TrainingLevel*6
    atk := g.TrainingBaseAtk + g.TrainingLevel/2
    enemy := Enemy{Name: "Hater d'entrainement", Type: enemyHater, MaxHP: hp, HP: hp, Attack: atk, CritTimer: 3, Style: "Troll"}
    if g.fightSolo(reader, enemy, battleOptions{
        AllowBet:     true,
        Intro:        []string{"Un hater veut tester ta concentration."},
        Victory:      []string{"Ton souffle gagne en puissance."},
        Defeat:       []string{"Les haters ricanent. Continue de t'entrainer."},
        RewardXP:     24,
        RewardGold:   5,
        RewardBetPts: 1,
    }) {
        g.TrainingLevel++
        g.TrainingBaseHP += 2
        g.TrainingBaseAtk++
        active := g.active()
        active.MaxHP += 5
        active.HP += 5
        if active.HP > active.MaxHP {
            active.HP = active.MaxHP
        }
        fmt.Printf("%s gagne en endurance (HP max %d).\n", active.Name, active.MaxHP)
        g.rewardMaterial(active)
        g.autoSave()
    }
}

// Combat de farm pour recolter or et XP
func (g *Game) farm(reader *bufio.Reader) {
    fmt.Println("\n=== Farm d'EXP ===")
    hp := 70 + g.FarmLevel*12
    atk := 8 + g.FarmLevel
    enemy := Enemy{Name: "Gardien repetitif", Type: enemyFarm, MaxHP: hp, HP: hp, Attack: atk, CritTimer: 3, Style: "Loop"}
    if g.fightSolo(reader, enemy, battleOptions{
        AllowEscape: true,
        Intro:       []string{"Un adversaire sans histoire te barre la route."},
        Victory:     []string{"Tu grappilles quelques fans et materiaux."},
        RewardXP:    15,
        RewardGold:  3,
    }) {
        g.FarmLevel++
        g.rewardMaterial(g.active())
        fmt.Println("Les adversaires de farm deviennent plus coriaces." )
        g.autoSave()
    }
}

// Choix ou creation d'un profil de sauvegarde
func promptProfile(sm *SaveManager, reader *bufio.Reader) (string, *SaveState) {
    for {
        profiles, err := sm.list()
        if err != nil {
            fmt.Println("Impossible de lister les profils:", err)
        }
        banner("Profils")
        if len(profiles) == 0 {
            fmt.Println("Aucun profil. Entrez un nom pour commencer:")
            name := read(reader)
            if name == "" {
                fmt.Println("Nom vide.")
                continue
            }
            return name, nil
        }
        for i, name := range profiles {
            fmt.Printf("%d) %s\n", i+1, name)
        }
        fmt.Println("0) Creer un nouveau profil")
        fmt.Print("Choix: ")
        choice, err := strconv.Atoi(read(reader))
        if err != nil || choice < 0 || choice > len(profiles) {
            fmt.Println("Choix invalide.")
            continue
        }
        if choice == 0 {
            fmt.Print("Nom du nouveau profil: ")
            name := read(reader)
            if name == "" {
                fmt.Println("Nom vide.")
                continue
            }
            return name, nil
        }
        name := profiles[choice-1]
        state, err := sm.load(name)
        if err != nil {
            fmt.Println("Lecture impossible:", err)
            continue
        }
        fmt.Printf("Profil '%s' charge (derniere sauvegarde %s).\n", name, state.Timestamp.Format(time.RFC1123))
        return name, state
    }
}

// Permet de changer de personnage jouable
func (g *Game) chooseCharacter(reader *bufio.Reader) {
    fmt.Println("\n=== Choix de personnage ===")
    for i, ch := range g.Characters {
        status := "disponible"
        if !ch.Unlocked {
            status = "verrouille"
        }
        fmt.Printf("%d) %s [%s]\n", i+1, ch.Name, status)
    }
    fmt.Print("Choix (0 annuler): ")
    choice, err := strconv.Atoi(read(reader))
    if g.consumeMenuReturn() {
        return
    }
    if err != nil || choice <= 0 || choice > len(g.Characters) {
        fmt.Println("Aucun changement.")
        return
    }
    if !g.Characters[choice-1].Unlocked {
        fmt.Println("Ce personnage n'est pas encore disponible.")
        return
    }
    g.PlayerIndex = choice - 1
    fmt.Printf("Vous incarnez maintenant %s.\n", g.active().Name)
}

// Declenche la prochaine etape du scenario
// Boucle principale du jeu
func (g *Game) runNextStory(reader *bufio.Reader) {
    switch g.StoryStage {
    case stagePrologue:
        g.prologue(reader)
    case stageArtists:
        g.artistHub(reader)
    case stageMacron:
        g.macronMission(reader)
    case stageLabel:
        g.labelFinal(reader)
    case stageFinish:
        fmt.Println("L'histoire principale est terminee. Continuez a jouer librement !")
    }
}

func (g *Game) run(reader *bufio.Reader) {
    setActiveGame(g)
    defer setActiveGame(nil)
    g.menuReturnRequested = false

    if g.StoryStage == stagePrologue {
        g.prologue(reader)
    }
    for {
        banner("Menu principal")
        active := g.active()
        fmt.Printf("Profil: %s | Or: %d | Perso: %s | Points de mise: %d\n", g.profile, g.Gold, active.Name, active.BetPts)
        fmt.Println("1) Continuer l'histoire")
        fmt.Println("2) Entrainement")
        fmt.Println("3) Farm d'EXP")
        fmt.Println("4) Statistiques")
        fmt.Println("5) Marchand")
        if g.CraftUnlocked {
            fmt.Println("6) Forgeron / Craft")
        } else {
            fmt.Println("6) Forgeron / Craft (verrouille)")
        }
        fmt.Println("7) Changer de personnage")
        fmt.Println("8) Sauvegarder")
        fmt.Println("9) Quitter")
        fmt.Print("Choix: ")
        choice := read(reader)
        if g.consumeMenuReturn() {
            continue
        }
        switch choice {
        case "1":
            g.runNextStory(reader)
        case "2":
            g.training(reader)
        case "3":
            g.farm(reader)
        case "4":
            active.printStats()
        case "5":
            g.handleMerchant(reader)
        case "6":
            g.handleCraft(reader)
        case "7":
            g.chooseCharacter(reader)
        case "8":
            g.autoSave()
        case "9":
            g.autoSave()
            fmt.Println("Merci d'avoir defendu la musique libre !")
            return
        default:
            fmt.Println("Choix invalide.")
        }
    }
}



// Point d'entree du programme
func main() {
    reader := bufio.NewReader(os.Stdin)
    sm := newSaveManager(saveDirName)
    profile, state := promptProfile(sm, reader)
    game := newGame(sm, profile, state)
    game.run(reader)
}

// Affiche l'etat des ennemis pendant un combat
func printEnemies(enemies []Enemy) {
    for i, e := range enemies {
        status := fmt.Sprintf("%d/%d HP", e.HP, e.MaxHP)
        if e.HP <= 0 {
            status = "KO"
        }
        fmt.Printf("  %d) %s (%s)\n", i+1, e.Name, status)
    }
}




