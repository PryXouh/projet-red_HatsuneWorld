package main

/*
   Ce programme implémente un jeu d'aventure textuel complet basé sur
   la Bible d'univers « Hatsune Miku et la Cassette Légendaire »【936002994787532†L36-L49】.

   L'objectif est de rester accessible pour des débutants (niveau Bachelor 1),
   sans librairies externes ni constructions Go avancées, tout en suivant
   l'histoire : la cassette légendaire a été volée par le label Pouler.fr,
   Miku doit parcourir plusieurs chapitres, rencontrer des artistes clés,
   affronter des ennemis (haters, sbires, rivales) et finalement battre
   les dirigeants du label pour restituer la vraie musique.

   Les mécaniques principales incluses :
   - Système de personnages jouables (Miku, Kaaris, Macron, Michael Jackson)
     avec PV et capacités spéciales conformes à la bible【936002994787532†L53-L79】.
   - Inventaire limité et système d'objets (potions, matériaux, livres de sort,
     équipements)【936002994787532†L100-L112】.
   - Marchand (disquaire) pour acheter objets et matériaux【936002994787532†L116-L127】.
   - Forgeron (ingénieur son) pour combiner matériaux et fabriquer des
     équipements ou disques empoisonnés【936002994787532†L132-L162】.
   - Recettes de craft (chapeau, tunique, bottes de scène) et disques
     empoisonnés avec effets spécifiques【936002994787532†L138-L160】.
   - Système de mise en combat : le joueur mise des points (x2/x3/x4),
     influençant la difficulté et les récompenses【936002994787532†L270-L277】.
   - Combat au tour par tour avec attaque de base, sort magique (Note
     explosive) et possibilité d'utiliser l'inventaire.
   - Progression en chapitres suivant la structure de la bible【936002994787532†L306-L313】 :
       1) Perte de la Cassette (prologue + tutoriel)
       2) Les Haters du Label (combats d’entraînement)
       3) Les Artistes Clés (rencontres MJ, Kaaris, Macron)
       4) Les Rivales (4 boss stylisés)
       5) Le Label Pouler.fr (combat final)

   Ce fichier est autonome : compilez avec `go build hatsune_game.go` et
   exécutez pour jouer. Les commentaires expliquent chaque étape pour
   faciliter la compréhension et la modification.
*/

import (
    "bufio"
    "fmt"
    "math/rand"
    "os"
    "strings"
    "time"
)

// -----------------------------------------------------------------------------
// Structures de données principales

// ItemType énumère les catégories d'objets possibles dans le jeu.
type ItemType int

const (
    Consumable ItemType = iota // consommable (potion)
    Equipment                 // équipement (bonus de PV)
    Special                   // objet spécial (livres, clés...)
    Material                  // matériau pour le craft
)

// Item représente un objet que le joueur peut acheter, utiliser ou porter.
type Item struct {
    Name        string    // nom de l'objet
    Type        ItemType  // type d'objet
    Description string    // description courte
    Price       int       // prix chez le marchand (0 si non achetable)
    Effect      func(p *Character) bool // fonction appelée lors de l'utilisation; retourne vrai si consommé
}

// Recipe définit une recette de craft : matériaux nécessaires et objet résultant.
type Recipe struct {
    Name   string   // nom de la recette/objet créé
    Inputs []string // liste de noms de matériaux requis
    Output Item     // objet créé
}

// Character modélise un personnage jouable (ou PNJ combattant) avec ses
// caractéristiques de base. Certaines capacités spéciales sont débloquées
// via l'histoire.
type Character struct {
    Name          string   // nom du personnage
    Class         string   // classe (Digital Idol, Force de la Rue, etc.)
    MaxHP         int      // points de vie maximum
    HP            int      // points de vie actuels
    MaxMana       int      // mana maximum
    Mana          int      // mana actuel
    Level         int      // niveau (augmente les stats)
    XP            int      // expérience accumulée
    Gold          int      // or disponible
    BetPts        int      // points de mise (utilisés pour parier avant un combat)
    Inventory     []Item   // inventaire du joueur (objets et matériaux)
    InventoryMax  int      // nombre maximum d'objets transportables
    Unlocked      bool     // indique si le personnage est disponible pour jouer
    HasNoteSpell  bool     // indique si le sort « Note explosive » est appris
    SpecialUsed   bool     // réinitialisé à chaque combat : vrai si capacité spéciale déjà utilisée
}

// Enemy représente un adversaire en combat. Les bosses et sbires sont
// également modélisés par cette structure simple.
type Enemy struct {
    Name     string // nom de l'ennemi
    HP       int    // points de vie actuels
    MaxHP    int    // points de vie maximum
    Attack   int    // dégâts de base infligés à chaque tour
    CritTimer int   // compteur de tours pour déclencher un coup critique
    Style    string // pour information (pop, rap, rock, classique)
}

// Game centralise l'état du jeu : personnages disponibles, joueur courant,
// inventaire, recettes, marchand, etc.
type Game struct {
    Player      *Character   // pointeur vers le personnage actuellement contrôlé
    Characters  []*Character // liste de tous les personnages jouables
    Merchant    []Item       // objets vendus chez le disquaire
    Materials   []Item       // matériaux vendus chez le marchand
    Recipes     []Recipe     // recettes de craft disponibles chez le forgeron
    rng         *rand.Rand   // générateur pseudo‑aléatoire
    StoryStage  int          // progression dans l'histoire (0 à 5)

    // Variables de progression pour les combats d'entraînement
    // Ces valeurs permettent de rendre la difficulté progressive et d'appliquer
    // un bonus au joueur à chaque victoire : la base PV des ennemis augmente
    // légèrement tandis que le joueur gagne des PV permanents.
    TrainingEnemyBaseHP    int
    TrainingEnemyBaseAttack int
}

// -----------------------------------------------------------------------------
// Initialisation du jeu

// NewGame crée une nouvelle partie en initialisant les personnages,
// marchandises, recettes et paramètres de base.
func NewGame() *Game {
    g := &Game{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
    // Création des personnages (seul Miku est débloqué au départ)
    miku := &Character{Name: "Hatsune Miku", Class: "Digital Idol", MaxHP: 80, HP: 80, MaxMana: 40, Mana: 40, Level: 1, XP: 0, Gold: 10, BetPts: 30, InventoryMax: 10, Unlocked: true, HasNoteSpell: false}
    kaaris := &Character{Name: "Kaaris", Class: "Force de la Rue", MaxHP: 120, HP: 120, MaxMana: 30, Mana: 30, Level: 1, XP: 0, Gold: 0, BetPts: 0, InventoryMax: 10, Unlocked: false}
    macron := &Character{Name: "Macron", Class: "Stratège Présidentiel", MaxHP: 100, HP: 100, MaxMana: 35, Mana: 35, Level: 1, XP: 0, Gold: 0, BetPts: 0, InventoryMax: 10, Unlocked: false}
    mj := &Character{Name: "Michael Jackson", Class: "Roi de la Pop", MaxHP: 100, HP: 100, MaxMana: 35, Mana: 35, Level: 1, XP: 0, Gold: 0, BetPts: 0, InventoryMax: 10, Unlocked: false}
    g.Characters = []*Character{miku, kaaris, macron, mj}
    g.Player = miku
    g.StoryStage = 0

    // Valeurs initiales pour l'entraînement : ennemis peu résistants
    g.TrainingEnemyBaseHP = 20
    g.TrainingEnemyBaseAttack = 5

    // Définition des potions et objets spéciaux vendus chez le marchand
    g.Merchant = []Item{
        // Potion de vie : soigne 50 PV【936002994787532†L100-L112】
        {Name: "Potion de vie", Type: Consumable, Description: "Soigne 50 PV", Price: 3, Effect: func(p *Character) bool {
            heal := 50
            if p.HP+heal > p.MaxHP {
                p.HP = p.MaxHP
            } else {
                p.HP += heal
            }
            fmt.Println("Vous buvez une potion de vie et récupérez des PV.")
            return true
        }},
        // Potion empoisonnée : utilisée seulement pour fabriquer des disques empoisonnés
        {Name: "Potion empoisonnée", Type: Consumable, Description: "À combiner pour créer un disque empoisonné", Price: 6, Effect: func(p *Character) bool {
            fmt.Println("Cette potion doit être combinée avec un matériau au forgeron.")
            return false
        }},
        // Potion d'énergie : rend 20 points de mana【936002994787532†L100-L112】
        {Name: "Potion d'énergie", Type: Consumable, Description: "Rend 20 points de mana", Price: 5, Effect: func(p *Character) bool {
            if p.Mana+20 > p.MaxMana {
                p.Mana = p.MaxMana
            } else {
                p.Mana += 20
            }
            fmt.Println("Vous buvez une potion d'énergie et récupérez de la mana.")
            return true
        }},
        // Livre de Sort : Note explosive : permet d'apprendre le sort magique
        {Name: "Livre de Sort : Note explosive", Type: Special, Description: "Permet d'apprendre le sort Note explosive", Price: 25, Effect: func(p *Character) bool {
            if p.HasNoteSpell {
                fmt.Println("Vous maîtrisez déjà le sort Note explosive.")
                return false
            }
            p.HasNoteSpell = true
            fmt.Println("Vous avez appris le sort Note explosive !")
            return true
        }},
        // Augmentation d'inventaire : +10 emplacements (max 3 fois)【936002994787532†L195-L206】
        {Name: "Augmentation d’inventaire", Type: Special, Description: "+10 emplacements d'inventaire", Price: 30, Effect: func(p *Character) bool {
            if p.InventoryMax >= 40 {
                fmt.Println("Votre inventaire est déjà au maximum.")
                return false
            }
            p.InventoryMax += 10
            fmt.Println("Vous augmentez la capacité de votre sacoche à vinyles !")
            return true
        }},
    }
    // Matériaux vendus chez le disquaire (pour craft)
    g.Materials = []Item{
        {Name: "Sample de Loup", Type: Material, Description: "Matériau pour disque", Price: 4},
        {Name: "Partition de Troll", Type: Material, Description: "Matériau pour disque", Price: 7},
        {Name: "Câble de Sanglier", Type: Material, Description: "Matériau pour disque", Price: 3},
        {Name: "Plume de Corbeau", Type: Material, Description: "Matériau pour disque", Price: 1},
    }
    // Ajout des matériaux au marchand
    g.Merchant = append(g.Merchant, g.Materials...)
    // Définition des recettes de craft : équipements de scène et disques empoisonnés
    g.Recipes = []Recipe{
        // Chapeau de scène : +10 PV max【936002994787532†L132-L135】
        {Name: "Chapeau de scène", Inputs: []string{"Plume de Corbeau", "Câble de Sanglier"}, Output: Item{Name: "Chapeau de scène", Type: Equipment, Description: "+10 PV max", Price: 0, Effect: func(p *Character) bool {
            p.MaxHP += 10
            p.HP += 10
            fmt.Println("Vous équipez le Chapeau de scène, votre PV max augmente de 10 !")
            return true
        }}},
        // Tunique de scène : +25 PV max【936002994787532†L132-L135】
        {Name: "Tunique de scène", Inputs: []string{"Sample de Loup", "Sample de Loup", "Partition de Troll"}, Output: Item{Name: "Tunique de scène", Type: Equipment, Description: "+25 PV max", Price: 0, Effect: func(p *Character) bool {
            p.MaxHP += 25
            p.HP += 25
            fmt.Println("Vous équipez la Tunique de scène, votre PV max augmente de 25 !")
            return true
        }}},
        // Bottes de scène : +15 PV max【936002994787532†L132-L135】
        {Name: "Bottes de scène", Inputs: []string{"Sample de Loup", "Câble de Sanglier"}, Output: Item{Name: "Bottes de scène", Type: Equipment, Description: "+15 PV max", Price: 0, Effect: func(p *Character) bool {
            p.MaxHP += 15
            p.HP += 15
            fmt.Println("Vous équipez les Bottes de scène, votre PV max augmente de 15 !")
            return true
        }}},
        // Disques empoisonnés : effets spécifiques【936002994787532†L138-L160】
        {Name: "Disque de Loup Empoisonné", Inputs: []string{"Sample de Loup", "Potion empoisonnée"}, Output: Item{Name: "Disque de Loup Empoisonné", Type: Special, Description: "+10 dégâts contre haters", Price: 0}},
        {Name: "Disque de Troll Empoisonné", Inputs: []string{"Partition de Troll", "Potion empoisonnée"}, Output: Item{Name: "Disque de Troll Empoisonné", Type: Special, Description: "+15 dégâts contre sbires costauds", Price: 0}},
        {Name: "Disque de Sanglier Empoisonné", Inputs: []string{"Câble de Sanglier", "Potion empoisonnée"}, Output: Item{Name: "Disque de Sanglier Empoisonné", Type: Special, Description: "Ignore la défense d’un boss pendant 1 tour", Price: 0}},
        {Name: "Disque de Corbeau Empoisonné", Inputs: []string{"Plume de Corbeau", "Potion empoisonnée"}, Output: Item{Name: "Disque de Corbeau Empoisonné", Type: Special, Description: "Inflige poison pendant 2 tours", Price: 0}},
    }
    return g
}

// -----------------------------------------------------------------------------
// Fonctions utilitaires pour l'inventaire et les personnages

// findItem recherche un objet par nom (insensible à la casse) dans l'inventaire
// et retourne son indice, ou -1 s'il n'est pas trouvé.
func (p *Character) findItem(name string) int {
    for i, item := range p.Inventory {
        if strings.EqualFold(item.Name, name) {
            return i
        }
    }
    return -1
}

// removeItems enlève les objets dont les noms apparaissent dans names, en
// respectant les quantités requises. Renvoie true si tous les objets sont
// disponibles et retirés, false sinon.
func (p *Character) removeItems(names []string) bool {
    // Compter les occurrences requises
    needed := make(map[string]int)
    for _, n := range names {
        needed[strings.ToLower(n)]++
    }
    // Identifiez les indices à enlever
    toRemove := []int{}
    for i, item := range p.Inventory {
        lower := strings.ToLower(item.Name)
        if needed[lower] > 0 {
            needed[lower]--
            toRemove = append(toRemove, i)
        }
    }
    // Si certains objets manquent, annuler
    for _, count := range needed {
        if count > 0 {
            return false
        }
    }
    // Supprimer les éléments en partant de la fin
    for i := len(toRemove) - 1; i >= 0; i-- {
        idx := toRemove[i]
        p.Inventory = append(p.Inventory[:idx], p.Inventory[idx+1:]...)
    }
    return true
}

// addItem ajoute un objet à l'inventaire du personnage s'il reste de la place.
func (p *Character) addItem(item Item) bool {
    if len(p.Inventory) >= p.InventoryMax {
        fmt.Println("Votre sacoche à vinyles est déjà remplie.")
        return false
    }
    p.Inventory = append(p.Inventory, item)
    return true
}

// showStats affiche les statistiques du personnage et son inventaire.
func (p *Character) showStats() {
    fmt.Printf("\n=== Stats de %s ===\n", p.Name)
    fmt.Printf("Classe : %s\n", p.Class)
    fmt.Printf("Niveau : %d (XP : %d)\n", p.Level, p.XP)
    fmt.Printf("PV : %d / %d\n", p.HP, p.MaxHP)
    fmt.Printf("Mana : %d / %d\n", p.Mana, p.MaxMana)
    fmt.Printf("Or : %d\n", p.Gold)
    fmt.Printf("Points de mise : %d\n", p.BetPts)
    fmt.Printf("Inventaire (%d/%d) :\n", len(p.Inventory), p.InventoryMax)
    if len(p.Inventory) == 0 {
        fmt.Println("  (vide)")
    }
    for i, item := range p.Inventory {
        fmt.Printf("  %d. %s (%s)\n", i+1, item.Name, item.Description)
    }
}

// chooseCharacter permet au joueur de sélectionner un personnage disponible.
// Le pointeur Player dans Game est mis à jour en conséquence.
func (g *Game) chooseCharacter(reader *bufio.Reader) {
    fmt.Println("\n=== Sélection de personnage ===")
    for i, c := range g.Characters {
        status := "(débloqué)"
        if !c.Unlocked {
            status = "(verrouillé)"
        }
        fmt.Printf("%d. %s %s\n", i+1, c.Name, status)
    }
    fmt.Print("Choisissez le numéro du personnage à incarner (0 pour revenir) : ")
    var choice int
    fmt.Fscanln(reader, &choice)
    if choice <= 0 || choice > len(g.Characters) {
        return
    }
    sel := g.Characters[choice-1]
    if !sel.Unlocked {
        fmt.Println("Ce personnage n'est pas encore disponible.")
        return
    }
    g.Player = sel
    fmt.Printf("Vous incarnez désormais %s.\n", sel.Name)
}

// levelUpIfNeeded augmente le niveau du personnage s'il a atteint 100 XP et
// réinitialise la barre d'expérience. Chaque niveau apporte +5 PV max et
// +5 mana max, et régénère totalement PV et mana【936002994787532†L288-L293】.
func (p *Character) levelUpIfNeeded() {
    for p.XP >= 100 {
        p.XP -= 100
        p.Level++
        p.MaxHP += 5
        p.MaxMana += 5
        p.HP = p.MaxHP
        p.Mana = p.MaxMana
        fmt.Printf("\nFélicitations ! %s passe au niveau %d. PV et Mana augmentent.\n", p.Name, p.Level)
    }
}

// resurrectIfNeeded réanime le personnage s'il est mort. Selon la bible,
// lorsqu'un artiste tombe à 0 PV, il revient à 50 % de ses PV grâce à ses fans【936002994787532†L280-L283】.
func (p *Character) resurrectIfNeeded() {
    if p.HP <= 0 {
        fmt.Printf("\n💫 %s tombe... mais ses fans le relèvent à 50 %% PV !\n", p.Name)
        p.HP = p.MaxHP / 2
    }
}

// resetSpecialUsage réinitialise le compteur d'utilisation de capacité spéciale
// avant un nouveau combat.
func (p *Character) resetSpecialUsage() {
    p.SpecialUsed = false
}

// -----------------------------------------------------------------------------
// Combats génériques

// battleOptions définit les paramètres variables selon le type de combat.
// Utilisé pour spécialiser les combats de tutoriel, d'entraînement ou de boss.
type battleOptions struct {
    AllowBet      bool // autoriser la mise en début de combat
    BetMultiplier int  // multiplicateur de difficulté (x2, x3, x4) pour l'entraînement
    AllowEscape   bool // autoriser la fuite
    EnemyDesc     string // description supplémentaire pour l'ennemi
    RewardXP      int  // gain en XP après victoire
    RewardGold    int  // gain en or après victoire
    RewardBet     int  // gain en points de mise après victoire (pour entraînement)
    IsBoss        bool // indique un combat de boss (affichage différent)
    UseDiscEffect bool // les disques empoisonnés appliquent des effets particuliers
}

// fight lance un combat entre le joueur courant et l'ennemi passé en paramètre.
// Les options permettent d'adapter la difficulté, la mise, l'utilisation des
// disques et les récompenses. Cette fonction est utilisée pour tous les
// affrontements (tutoriel, entraînement, histoire, bosses).
func (g *Game) fight(enemy Enemy, opts battleOptions, reader *bufio.Reader) bool {
    p := g.Player
    p.resetSpecialUsage()
    // Mise éventuelle
    bet := 0
    if opts.AllowBet {
        if p.BetPts <= 0 {
            fmt.Println("Vous n'avez pas de points de mise. Le combat commence sans pari.")
        } else {
            fmt.Printf("Points de mise disponibles : %d\n", p.BetPts)
            fmt.Println("Choisissez une mise : 1) x2  2) x3  3) x4")
            var choice int
            fmt.Fscanln(reader, &choice)
            switch choice {
            case 1:
                bet = 2
            case 2:
                bet = 3
            case 3:
                bet = 4
            default:
                bet = 2
            }
            // Les points de mise ne sont pas retirés immédiatement : ils sont
            // perdus en cas de défaite, gagnés en cas de victoire.
        }
        // Adapter la difficulté et les récompenses selon la mise
        if bet > 0 {
            enemy.HP = enemy.MaxHP * bet
            enemy.MaxHP = enemy.HP
            enemy.Attack = enemy.Attack * bet
            opts.RewardXP *= bet
            opts.RewardGold *= bet
            opts.RewardBet = bet
        }
    }
    // Paramètre de précision des attaques ennemies : plus la mise est haute,
    // moins l'ennemi rate (minimum 50 %).
    successRate := 80
    if bet > 0 {
        // Pour x2, x3, x4 on réduit successRate de 15 % par niveau au-delà de 2.
        successRate = 80 - (bet-2)*15
        if successRate < 50 {
            successRate = 50
        }
    }

    // Variables pour la gestion du poison via Disque de Corbeau
    poisonTurns := 0
    poisonDamage := 0

    // Boucle de combat jusqu'à ce qu'un camp tombe à 0 PV
    playerTurn := true
    for p.HP > 0 && enemy.HP > 0 {
        // Séparateur visuel pour rendre le journal de combat plus lisible
        fmt.Println("--------------------------------------------------")
        if playerTurn {
            // Tour du joueur : afficher état et options
            fmt.Printf("\n— Votre tour —\n")
            fmt.Printf("%s : %d/%d PV | %d/%d PM\n", p.Name, p.HP, p.MaxHP, p.Mana, p.MaxMana)
            fmt.Printf("%s : %d/%d PV\n", enemy.Name, enemy.HP, enemy.MaxHP)
            fmt.Println("1) Attaquer (8)")
            if p.HasNoteSpell {
                fmt.Println("2) Note explosive (18, 10 PM)")
            } else {
                fmt.Println("2) (Sort verrouillé — achetez le Livre au Marchand)")
            }
            // Capacité spéciale (une fois par combat)
            fmt.Println("3) Capacité spéciale")
            fmt.Println("4) Inventaire (utiliser potion, disque ou équipement)")
            if opts.AllowEscape {
                fmt.Println("5) Fuir (mettre fin au combat)")
            }
            fmt.Print("Choix : ")
            var action int
            fmt.Fscanln(reader, &action)
            switch action {
            case 1:
                // Attaque de base
                dmg := 8
                before := enemy.HP
                enemy.HP -= dmg
                if enemy.HP < 0 {
                    enemy.HP = 0
                }
                fmt.Printf("➡️  Coup de poing ! %s %d → %d  (-%d PV)\n", enemy.Name, before, enemy.HP, before-enemy.HP)
            case 2:
                // Sort Note explosive
                if !p.HasNoteSpell || p.Mana < 10 {
                    fmt.Println("⛔ Sort indisponible.")
                    continue
                }
                p.Mana -= 10
                dmg := 18
                before := enemy.HP
                enemy.HP -= dmg
                if enemy.HP < 0 {
                    enemy.HP = 0
                }
                fmt.Printf("🎶 Note explosive ! %s %d → %d  (-%d PV) | PM -10\n", enemy.Name, before, enemy.HP, before-enemy.HP)
            case 3:
                // Capacité spéciale du personnage courant
                if p.SpecialUsed {
                    fmt.Println("Vous avez déjà utilisé votre capacité spéciale dans ce combat.")
                    continue
                }
                // Appliquer un effet différent selon le personnage
                switch p.Name {
                case "Hatsune Miku":
                    // Miku : récupère 20 Mana et 10 PV comme boost scénique
                    beforeHP := p.HP
                    beforeMana := p.Mana
                    heal := 10
                    manaGain := 20
                    p.HP += heal
                    if p.HP > p.MaxHP {
                        p.HP = p.MaxHP
                    }
                    p.Mana += manaGain
                    if p.Mana > p.MaxMana {
                        p.Mana = p.MaxMana
                    }
                    fmt.Printf("✨ Miku entonne un solo : PV %d → %d (+%d) | PM %d → %d (+%d)\n", beforeHP, p.HP, p.HP-beforeHP, beforeMana, p.Mana, p.Mana-beforeMana)
                case "Kaaris":
                    // Kaaris : invoque le crew et inflige de gros dégâts instantanés
                    dmg := 25
                    before := enemy.HP
                    enemy.HP -= dmg
                    if enemy.HP < 0 {
                        enemy.HP = 0
                    }
                    fmt.Printf("🔥 Kaaris invoque son crew ! %s %d → %d  (-%d PV)\n", enemy.Name, before, enemy.HP, before-enemy.HP)
                case "Macron":
                    // Macron : discourt et affaiblit l'adversaire pour deux tours
                    enemy.Attack /= 2
                    enemy.CritTimer = 4 // reporter le prochain critique
                    fmt.Println("🗣️  Macron prononce un discours : l'adversaire est perturbé (attaque divisée par 2 pendant quelques tours).")
                case "Michael Jackson":
                    // MJ : Moonwalk, esquive le prochain coup et inflige des dégâts
                    dmg := 15
                    before := enemy.HP
                    enemy.HP -= dmg
                    if enemy.HP < 0 {
                        enemy.HP = 0
                    }
                    fmt.Printf("🌙 Moonwalk ! %s %d → %d  (-%d PV). Vous évitez la prochaine attaque ennemie.\n", enemy.Name, before, enemy.HP, before-enemy.HP)
                    // On indiquera un état d'esquive via un marqueur temporaire
                    enemy.CritTimer++ // repousser critique pour simuler esquive
                default:
                    fmt.Println("Capacité spéciale non définie pour ce personnage.")
                }
                p.SpecialUsed = true
            case 4:
                // Utiliser un objet de l'inventaire
                if len(p.Inventory) == 0 {
                    fmt.Println("Votre inventaire est vide.")
                    continue
                }
                fmt.Println("Inventaire :")
                for i, it := range p.Inventory {
                    fmt.Printf("  %d. %s (%s)\n", i+1, it.Name, it.Description)
                }
                fmt.Print("Sélectionnez un objet (0 pour annuler) : ")
                var idx int
                fmt.Fscanln(reader, &idx)
                if idx <= 0 || idx > len(p.Inventory) {
                    continue
                }
                item := p.Inventory[idx-1]
                // Disques empoisonnés : appliquer effet spécifique sur l'ennemi
                if strings.HasPrefix(item.Name, "Disque") {
                    switch item.Name {
                    case "Disque de Loup Empoisonné":
                        dmg := 10
                        before := enemy.HP
                        enemy.HP -= dmg
                        if enemy.HP < 0 {
                            enemy.HP = 0
                        }
                        fmt.Printf("💿 Disque de Loup : %s %d → %d  (-%d PV)\n", enemy.Name, before, enemy.HP, before-enemy.HP)
                    case "Disque de Troll Empoisonné":
                        dmg := 15
                        before := enemy.HP
                        enemy.HP -= dmg
                        if enemy.HP < 0 {
                            enemy.HP = 0
                        }
                        fmt.Printf("💿 Disque de Troll : %s %d → %d  (-%d PV)\n", enemy.Name, before, enemy.HP, before-enemy.HP)
                    case "Disque de Sanglier Empoisonné":
                        // Ignore la défense : double les dégâts de votre prochain coup
                        fmt.Println("💿 Disque de Sanglier : votre prochaine attaque ignorera la défense du boss !")
                        enemy.Attack = enemy.Attack // pas d'effet immédiat, placeholder
                    case "Disque de Corbeau Empoisonné":
                        // Poison : applique 5 dégâts par tour pendant 2 tours
                        poisonTurns = 2
                        poisonDamage = 5
                        fmt.Println("💿 Disque de Corbeau : l'ennemi est empoisonné pendant 2 tours !")
                    }
                    // Retirer l'objet une fois utilisé
                    p.Inventory = append(p.Inventory[:idx-1], p.Inventory[idx:]...)
                } else if item.Effect != nil {
                    // Appliquer l'effet de la potion ou de l'augmentation
                    consumed := item.Effect(p)
                    if consumed {
                        p.Inventory = append(p.Inventory[:idx-1], p.Inventory[idx:]...)
                    }
                } else {
                    fmt.Println("Cet objet ne peut pas être utilisé directement.")
                }
            case 5:
                if opts.AllowEscape {
                    fmt.Println("Vous fuyez le combat…")
                    return false
                }
                fmt.Println("Option invalide.")
                continue
            default:
                fmt.Println("Choix invalide.")
                continue
            }
            playerTurn = false
        } else {
            // Tour de l'ennemi
            fmt.Println("\n— Tour de l’ennemi —")
            // Appliquer poison si actif
            if poisonTurns > 0 {
                before := enemy.HP
                enemy.HP -= poisonDamage
                if enemy.HP < 0 {
                    enemy.HP = 0
                }
                poisonTurns--
                fmt.Printf("☠️  Le poison fait effet : %s %d → %d  (-%d PV)\n", enemy.Name, before, enemy.HP, before-enemy.HP)
            }
            // L'ennemi peut rater son attaque (probabilité inverse du successRate)
            if g.rng.Intn(100) > successRate {
                fmt.Println("🤞 L'ennemi rate son coup !")
            } else {
                dmg := enemy.Attack
                // Coup critique tous les 3 tours
                if enemy.CritTimer == 1 {
                    dmg *= 2
                    enemy.CritTimer = 3
                    fmt.Println("‼️  Coup critique x2 !")
                } else {
                    enemy.CritTimer--
                }
                before := p.HP
                p.HP -= dmg
                if p.HP < 0 {
                    p.HP = 0
                }
                fmt.Printf("💥 %s attaque : %s %d → %d  (-%d PV)\n", enemy.Name, p.Name, before, p.HP, before-p.HP)
            }
            playerTurn = true
        }
    }
    // Détermination du vainqueur
    if p.HP <= 0 {
        // Défaite : résurrection et pénalité éventuelle
        p.resurrectIfNeeded()
        if opts.AllowBet && bet > 0 {
            // Perte de la mise
            p.BetPts -= bet
            if p.BetPts < 0 {
                p.BetPts = 0
            }
            fmt.Printf("Vous perdez votre mise. Points de mise restants : %d\n", p.BetPts)
        }
        return false
    }
    // Victoire : distribution des récompenses
    fmt.Println("\n🏆 Victoire !")
    p.XP += opts.RewardXP
    p.Gold += opts.RewardGold
    if opts.AllowBet && bet > 0 {
        p.BetPts += opts.RewardBet
        fmt.Printf("Points de mise +%d (total : %d)\n", opts.RewardBet, p.BetPts)
    }
    // Level‑up éventuel
    p.levelUpIfNeeded()
    return true
}

// -----------------------------------------------------------------------------
// Système de craft (forgeron / ingénieur du son)

// handleCraft permet au joueur de sélectionner une recette et de la fabriquer
// si les ressources nécessaires sont disponibles.
func (g *Game) handleCraft(reader *bufio.Reader) {
    fmt.Println("\n=== Forgeron / Ingé son ===")
    fmt.Println("Recettes disponibles :")
    for i, r := range g.Recipes {
        fmt.Printf("%d) %s -> %s\n", i+1, strings.Join(r.Inputs, ", "), r.Name)
    }
    fmt.Println("0) Retour")
    fmt.Print("Choisissez une recette : ")
    var choice int
    fmt.Fscanln(reader, &choice)
    if choice <= 0 || choice > len(g.Recipes) {
        return
    }
    recipe := g.Recipes[choice-1]
    if !g.Player.removeItems(recipe.Inputs) {
        fmt.Println("Il vous manque des matériaux pour fabriquer cela.")
        return
    }
    // Ajout de l'objet créé à l'inventaire
    if g.Player.addItem(recipe.Output) {
        fmt.Printf("Vous avez fabriqué %s !\n", recipe.Name)
    }
}

// -----------------------------------------------------------------------------
// Marchand : achat d'objets et de matériaux

// handleMerchant gère l'interaction avec le disquaire (boutique).
func (g *Game) handleMerchant(reader *bufio.Reader) {
    fmt.Println("\n=== Disquaire ===")
    fmt.Println("Bienvenue dans ma boutique ! Que désirez-vous ?")
    for i, item := range g.Merchant {
        fmt.Printf("%d) %s (%s) - %d or\n", i+1, item.Name, item.Description, item.Price)
    }
    fmt.Println("0) Retour")
    fmt.Printf("Or disponible : %d\n", g.Player.Gold)
    fmt.Print("Choisissez un article à acheter : ")
    var choice int
    fmt.Fscanln(reader, &choice)
    if choice <= 0 || choice > len(g.Merchant) {
        return
    }
    item := g.Merchant[choice-1]
    if g.Player.Gold < item.Price {
        fmt.Println("Vous n’avez pas assez de fans pour payer (or insuffisant).")
        return
    }
    if g.Player.addItem(item) {
        g.Player.Gold -= item.Price
        fmt.Printf("Vous achetez %s.\n", item.Name)
    }
}

// -----------------------------------------------------------------------------
// Histoire et chapitres

// runPrologue raconte l'introduction et propose un combat tutoriel sans mise.
func (g *Game) runPrologue(reader *bufio.Reader) {
    if g.StoryStage > 0 {
        return
    }
    fmt.Println("\n=== Prologue : Perte de la Cassette ===")
    fmt.Println("La cassette légendaire, source de toute bonne musique, vient d’être volée.")
    fmt.Println("Le label Pouler.fr, qui contrôle 90 % du PIB musical mondial, la détient désormais【936002994787532†L36-L49】.")
    fmt.Println("Derrière cette organisation se cachent Mattieu Berger et Sylvain Bagland, bien décidés à étouffer la créativité.\n")
    fmt.Println("Hatsune Miku, idole digitale, jure de récupérer la cassette et de rendre la musique au public.")
    fmt.Println("Pour te préparer, tu vas affronter un hater en combat d’entraînement.")
    fmt.Print("Lancer un court combat tutoriel ? 1) Oui  2) Non : ")
    var ans int
    fmt.Fscanln(reader, &ans)
    if ans == 1 {
        // Combat tutoriel sans mise
        tutEnemy := Enemy{Name: "Hater (Tutoriel)", HP: 20, MaxHP: 20, Attack: 5, CritTimer: 3}
        opts := battleOptions{AllowBet: false, AllowEscape: false, RewardXP: 10, RewardGold: 3}
        g.fight(tutEnemy, opts, reader)
    }
    fmt.Println("\nLe prologue est terminé. Tu peux désormais avancer dans l'histoire ou t'entraîner.")
    g.StoryStage = 1
}

// runHatersStage correspond au chapitre « Les Haters du Label ». Le joueur
// affronte plusieurs haters pour gagner de l'expérience et des matériaux.
func (g *Game) runHatersStage(reader *bufio.Reader) {
    fmt.Println("\n=== Chapitre 2 : Les Haters du Label ===")
    fmt.Println("Le label envoie des fans toxiques te barrer la route. Affronte-les pour prouver ta valeur.")
    // Deux combats d'entraînement, difficulté croissante mais progressive
    for i := 1; i <= 2; i++ {
        fmt.Printf("\n— Combat contre un Hater %d —\n", i)
        // Les premiers ennemis ont moins de PV et de puissance
        baseHP := 20 + 5*(i-1)    // 20 puis 25 PV
        baseAtk := 5 + (i - 1)    // 5 puis 6 dégâts
        enemy := Enemy{Name: "Hater", HP: baseHP, MaxHP: baseHP, Attack: baseAtk, CritTimer: 3}
        opts := battleOptions{AllowBet: true, AllowEscape: false, RewardXP: 15, RewardGold: 5}
        g.fight(enemy, opts, reader)
        // Récompense : un matériau aléatoire
        mat := g.Materials[g.rng.Intn(len(g.Materials))]
        g.Player.addItem(mat)
        fmt.Printf("Vous trouvez un %s dans les décombres.\n", mat.Name)
        // Progession : le joueur gagne 10 PV max et l'ennemi progresse de 5 PV pour les prochains combats
        g.Player.MaxHP += 10
        g.Player.HP += 10
        if g.Player.HP > g.Player.MaxHP {
            g.Player.HP = g.Player.MaxHP
        }
        g.TrainingEnemyBaseHP += 5
        fmt.Println("Votre endurance augmente (+10 PV max) et les ennemis deviennent un peu plus résistants (+5 PV).")
    }
    fmt.Println("\nAprès avoir repoussé les haters, vous avez gagné de l'expérience, de l'or et des matériaux.")
    fmt.Println("Vous pouvez maintenant rencontrer des artistes légendaires qui vous aideront dans votre quête.")
    g.StoryStage = 2
}

// askQuestion pose une question de quiz et retourne vrai si la réponse est correcte.
func askQuestion(reader *bufio.Reader, question string, correct string) bool {
    fmt.Println(question)
    fmt.Print("Votre réponse : ")
    ans, _ := reader.ReadString('\n')
    ans = strings.TrimSpace(strings.ToLower(ans))
    return ans == strings.ToLower(correct)
}

// meetMichaelJackson met en scène la rencontre avec Michael Jackson. Un
// mini-jeu (question) permet de débloquer le personnage et d'obtenir le
// Gant Légendaire.
func (g *Game) meetMichaelJackson(reader *bufio.Reader) {
    if g.Characters[3].Unlocked {
        return
    }
    fmt.Println("\n=== Rencontre avec Michael Jackson ===")
    fmt.Println("Sur ta route, le Roi de la Pop apparaît, triste de l'état de la musique actuelle.")
    fmt.Println("Il t'interroge pour voir si tu connais la culture musicale.")
    question := "Quel est le titre du pas iconique exécuté par Michael Jackson lors du 25e anniversaire de Motown (Moonwalk/Robot/Shuffle) ?"
    if askQuestion(reader, question, "moonwalk") {
        fmt.Println("Correct ! MJ est impressionné par ta culture et décide de t'aider.")
        // Débloquer MJ et offrir le Gant Légendaire (+25 PV max)
        g.Characters[3].Unlocked = true
        // Ajouter le Gant à l'inventaire du joueur
        glove := Item{Name: "Gant Légendaire", Type: Equipment, Description: "+25 PV max", Price: 0, Effect: func(p *Character) bool {
            p.MaxHP += 25
            p.HP += 25
            fmt.Println("Vous équipez le Gant Légendaire, votre PV max augmente de 25 !")
            return true
        }}
        g.Player.addItem(glove)
        fmt.Println("Michael Jackson rejoint votre équipe !")
        g.Characters[3].Gold = 0
        g.Characters[3].BetPts = 0
    } else {
        fmt.Println("Mauvaise réponse. MJ part déçu, mais reviendra peut-être plus tard.")
    }
}

// meetKaaris met en scène la rencontre avec Kaaris. Le joueur doit affronter
// un crew de la rue pour obtenir le Pouvoir d'Invocation et débloquer
// Kaaris comme personnage jouable.
func (g *Game) meetKaaris(reader *bufio.Reader) {
    if g.Characters[1].Unlocked {
        return
    }
    fmt.Println("\n=== Rencontre avec Kaaris ===")
    fmt.Println("Kaaris surgit de la cité et te met au défi : vaincs son crew pour gagner son respect.")
    // Combat spécial contre un sbire costaud (utiliser Disque de Troll pour avantage)
    // Équipe réduite pour rendre le combat accessible
    enemy := Enemy{Name: "Crew de Kaaris", HP: 50, MaxHP: 50, Attack: 10, CritTimer: 3}
    opts := battleOptions{AllowBet: false, AllowEscape: false, RewardXP: 20, RewardGold: 10, UseDiscEffect: true}
    if g.fight(enemy, opts, reader) {
        fmt.Println("Kaaris est impressionné par ta force.")
        g.Characters[1].Unlocked = true
        // Pouvoir d'invocation : objet spécial permettant d'invoquer le crew une fois par combat
        crewPower := Item{Name: "Pouvoir d’Invocation", Type: Special, Description: "Invoque le crew de Kaaris", Price: 0, Effect: func(p *Character) bool {
            // Effet : infliger 25 dégâts instantanés
            fmt.Println("Vous invoquez le crew de Kaaris et infligez 25 dégâts supplémentaires !")
            return true
        }}
        g.Player.addItem(crewPower)
        fmt.Println("Kaaris rejoint votre équipe !")
    } else {
        fmt.Println("Kaaris n'est pas convaincu. Retente ta chance plus tard.")
    }
}

// meetMacron propose un quiz de culture générale (3 questions) pour obtenir
// le Pass Présidentiel et débloquer Macron.
func (g *Game) meetMacron(reader *bufio.Reader) {
    if g.Characters[2].Unlocked {
        return
    }
    fmt.Println("\n=== Rencontre avec Macron ===")
    fmt.Println("Sur ta route se dresse le Président, gardien du label. Il teste ta culture générale.")
    questions := []struct{ q, a string }{
        {"En quelle année la Révolution française a-t-elle débuté ?", "1789"},
        {"Quelle est la devise de la République française (3 mots) ?", "liberté egalité fraternité"},
        {"Qui a composé La Marseillaise ?", "rouget de lisle"},
    }
    correctAnswers := 0
    for _, qa := range questions {
        if askQuestion(reader, qa.q, qa.a) {
            fmt.Println("Bonne réponse.")
            correctAnswers++
        } else {
            fmt.Println("Mauvaise réponse.")
        }
    }
    if correctAnswers == len(questions) {
        fmt.Println("Macron reconnaît ta culture et t'accorde un Pass Présidentiel.")
        g.Characters[2].Unlocked = true
        pass := Item{Name: "Pass Présidentiel", Type: Special, Description: "Permet d’accéder au QG du label", Price: 0, Effect: func(p *Character) bool {
            fmt.Println("Vous utilisez le Pass Présidentiel pour ouvrir une porte... rien ne se passe pour le moment.")
            return false
        }}
        g.Player.addItem(pass)
        fmt.Println("Macron rejoint votre équipe (peut affaiblir les adversaires) !")
    } else {
        fmt.Println("Macron t'invite à réviser et à revenir plus tard.")
    }
}

// runArtistsStage exécute le chapitre « Les Artistes Clés » en rencontrant
// successivement Michael Jackson, Kaaris et Macron. Chaque rencontre peut
// débloquer un personnage et un objet spécial.
func (g *Game) runArtistsStage(reader *bufio.Reader) {
    fmt.Println("\n=== Chapitre 3 : Les Artistes Clés ===")
    fmt.Println("Tu vas croiser des artistes légendaires. Réussis leurs épreuves pour qu'ils t'aident.")
    // Rencontre avec MJ
    g.meetMichaelJackson(reader)
    // Rencontre avec Kaaris
    g.meetKaaris(reader)
    // Rencontre avec Macron
    g.meetMacron(reader)
    fmt.Println("\nAprès avoir rencontré ces artistes, ton équipe s'agrandit et tu obtiens de précieux objets.")
    g.StoryStage = 3
}

// runRivalesStage conduit le joueur à affronter les 4 rivales du label.
func (g *Game) runRivalesStage(reader *bufio.Reader) {
    fmt.Println("\n=== Chapitre 4 : Les Rivales ===")
    fmt.Println("Les quatre rivales du label t'attendent. Chacune incarne un style musical et possède une attaque spéciale.")
    // Définir les rivales et leurs caractéristiques
    rivales := []Enemy{
        // Les PV et dégâts sont réduits pour une progression plus douce
        {Name: "Rivale Pop", HP: 60, MaxHP: 60, Attack: 8, CritTimer: 3, Style: "Pop"},
        {Name: "Rivale Rap", HP: 70, MaxHP: 70, Attack: 9, CritTimer: 3, Style: "Rap"},
        {Name: "Rivale Rock", HP: 80, MaxHP: 80, Attack: 10, CritTimer: 3, Style: "Rock"},
        {Name: "Rivale Classique", HP: 90, MaxHP: 90, Attack: 11, CritTimer: 3, Style: "Classique"},
    }
    for _, boss := range rivales {
        fmt.Printf("\nTu affrontes %s. Prépare-toi !\n", boss.Name)
        // Permettre au joueur de choisir son personnage avant chaque boss
        g.chooseCharacter(reader)
        opts := battleOptions{AllowBet: false, AllowEscape: false, RewardXP: 30, RewardGold: 15, IsBoss: true, UseDiscEffect: true}
        g.fight(boss, opts, reader)
        // Après chaque victoire, offrir un matériau rare ou des potions
        rewardMat := g.Materials[g.rng.Intn(len(g.Materials))]
        g.Player.addItem(rewardMat)
        fmt.Printf("Vous récupérez %s comme trophée.\n", rewardMat.Name)
    }
    fmt.Println("\nLes rivales sont vaincues. Le chemin vers le label est désormais ouvert.")
    g.StoryStage = 4
}

// runFinalStage affronte le boss final du label (Mattieu Berger & Sylvain Bagland)
// et conclut l'histoire.
func (g *Game) runFinalStage(reader *bufio.Reader) {
    fmt.Println("\n=== Chapitre 5 : Le Label Pouler.fr ===")
    fmt.Println("Le moment est venu d'affronter les dirigeants de Pouler.fr et de récupérer la cassette légendaire.")
    // Préparation : vérifie que le joueur possède le Pass Présidentiel
    if g.Player.findItem("Pass Présidentiel") < 0 {
        fmt.Println("Vous avez besoin du Pass Présidentiel pour entrer au QG. Retournez voir Macron.")
        return
    }
    // Combat final : un ennemi très puissant
    // Boss final légèrement réduit pour éviter un pic de difficulté trop abrupt
    finalBoss := Enemy{Name: "Mattieu & Sylvain", HP: 150, MaxHP: 150, Attack: 14, CritTimer: 3, Style: "Boss"}
    opts := battleOptions{AllowBet: false, AllowEscape: false, RewardXP: 100, RewardGold: 50, IsBoss: true, UseDiscEffect: true}
    // Permettre au joueur de choisir son personnage pour le combat final
    g.chooseCharacter(reader)
    if g.fight(finalBoss, opts, reader) {
        fmt.Println("\n🎉 Félicitations ! Vous avez vaincu les dirigeants du label Pouler.fr et récupéré la Cassette Légendaire.")
        fmt.Println("La vraie musique appartient aux artistes et au public, pas aux labels !【936002994787532†L327-L332】")
        fmt.Println("Fin du jeu. Merci d’avoir joué !\n")
        g.StoryStage = 5
    } else {
        fmt.Println("Les dirigeants ont eu raison de vous. Réessayez lorsque vous serez prêt.")
    }
}

// runNextStory déclenche le chapitre suivant en fonction de la progression
// actuelle. Si tous les chapitres sont terminés, un message final est affiché.
func (g *Game) runNextStory(reader *bufio.Reader) {
    switch g.StoryStage {
    case 0:
        g.runPrologue(reader)
    case 1:
        g.runHatersStage(reader)
    case 2:
        g.runArtistsStage(reader)
    case 3:
        g.runRivalesStage(reader)
    case 4:
        g.runFinalStage(reader)
    case 5:
        fmt.Println("\nVous avez déjà terminé l'histoire complète. Profitez du jeu librement !")
    default:
        fmt.Println("Une erreur s'est produite dans la progression de l'histoire.")
    }
}

// showMainMenu affiche les options principales disponibles à tout moment.
func showMainMenu() {
    fmt.Println("\n===== Menu Principal =====")
    fmt.Println("1) Continuer l'histoire")
    fmt.Println("2) Entraînement (mises x2/x3/x4)")
    fmt.Println("3) Statistiques du personnage")
    fmt.Println("4) Marchand (disquaire)")
    fmt.Println("5) Forgeron / Craft")
    fmt.Println("6) Changer de personnage")
    fmt.Println("7) Quitter")
    fmt.Print("Choix : ")
}

// run lance la boucle principale du jeu, en proposant le prologue puis
// les différentes options jusqu'à ce que le joueur quitte.
func (g *Game) run() {
    reader := bufio.NewReader(os.Stdin)
    // Exécution du prologue dès le lancement
    g.runPrologue(reader)
    for {
        showMainMenu()
        var choice int
        fmt.Fscanln(reader, &choice)
        switch choice {
        case 1:
            g.runNextStory(reader)
        case 2:
            // Combat d'entraînement avec mise et progression dynamique
            g.runTraining(reader)
        case 3:
            g.Player.showStats()
        case 4:
            g.handleMerchant(reader)
        case 5:
            g.handleCraft(reader)
        case 6:
            g.chooseCharacter(reader)
        case 7:
            fmt.Println("Au revoir et à bientôt !")
            return
        default:
            fmt.Println("Choix invalide. Veuillez réessayer.")
        }
    }
}

// runTraining lance un combat d'entraînement contre un hater dont la difficulté
// augmente progressivement. En cas de victoire, le joueur gagne 10 PV max et
// les ennemis gagnent 5 PV pour la prochaine séance. Les mises x2/x3/x4
// sont toujours possibles et ajustent également les récompenses.
func (g *Game) runTraining(reader *bufio.Reader) {
    fmt.Println("\n=== Séance d'entraînement ===")
    // Générer un ennemi basé sur les valeurs de progression actuelles
    baseHP := g.TrainingEnemyBaseHP
    baseAtk := g.TrainingEnemyBaseAttack
    enemy := Enemy{Name: "Hater", HP: baseHP, MaxHP: baseHP, Attack: baseAtk, CritTimer: 3}
    opts := battleOptions{AllowBet: true, AllowEscape: false, RewardXP: 10, RewardGold: 5}
    // Combattre ; si victoire, ajuster les statistiques
    if g.fight(enemy, opts, reader) {
        // Augmenter les PV max du joueur de 10 et soigner proportionnellement
        g.Player.MaxHP += 10
        g.Player.HP += 10
        if g.Player.HP > g.Player.MaxHP {
            g.Player.HP = g.Player.MaxHP
        }
        fmt.Printf("🎁 Votre endurance augmente : PV max +10 (nouveau max : %d)\n", g.Player.MaxHP)
        // Augmenter la difficulté de l'entraînement
        g.TrainingEnemyBaseHP += 5
        fmt.Println("Les ennemis d'entraînement deviennent un peu plus résistants (+5 PV).")
    } else {
        fmt.Println("Continuez à vous entraîner pour progresser.")
    }
}

// main est le point d'entrée du programme. Il crée une nouvelle partie et
// appelle run() pour lancer l'interface utilisateur.
func main() {
    game := NewGame()
    game.run()
}
