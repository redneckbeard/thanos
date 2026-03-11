# ============================================================
#   QUEST FOR THE GOLDEN HASH — a thanos showcase
# ============================================================

$turn = 0

MAX_HP = 100
CRIT_MULT = 2

# -- Struct ---------------------------------------------------
Loot = Struct.new(:name, :value) do
  def to_s
    "#{name} (#{value}g)"
  end
end

# -- Module with class methods and constants ------------------
module GameUtils
  Scale = 1.5

  def self.clamp(val, low, high)
    return low if val < low
    return high if val > high
    val
  end

  def self.banner(text)
    line = "=" * 50
    puts line
    puts text.center(50)
    puts line
  end

  def self.pseudo_rand(seed)
    ((seed * 1103515245 + 12345) % 32768).abs
  end
end

# -- Hero with Comparable mixin ------------------------------
class Hero
  include Comparable
  @@count = 0

  attr_accessor :name, :hp, :attack
  attr_reader :level, :xp, :role

  def initialize(name, hp, attack, role)
    @name = name
    @hp = hp
    @attack = attack
    @role = role
    @level = 1
    @xp = 0
    @inventory = []
    @inventory << Loot.new("Rusty Sword", 5)
    @@count += 1
  end

  def self.count
    @@count
  end

  def <=>(other)
    @level <=> other.level
  end

  def alive?
    @hp > 0
  end

  def role_title
    case @role
    when "warrior"
      "#{@name} the Brave"
    when "mage"
      "#{@name} the Wise"
    when "rogue"
      "#{@name} the Shadow"
    else
      @name
    end
  end
  alias title role_title

  def gain_xp(amount)
    @xp += amount
    puts "  #{@name} gains #{amount} XP (total: #{@xp})"
    level_up if @xp >= @level * 100
  end

  def level_up
    @level += 1
    @attack += 3
    @hp = MAX_HP
    puts "  *** #{@name} reached level #{@level}! ATK is now #{@attack} ***"
  end

  def loot(item)
    @inventory << item
    puts "  #{@name} picks up #{item}"
  end

  def inventory_value
    @inventory.map { |i| i.value }.sum
  end

  def show_inventory
    val = @inventory.map { |i| i.value }.sum
    cnt = @inventory.count
    puts "  Inventory (#{cnt} items, #{val}g total):"
    @inventory.each_with_index do |item, i|
      num = i + 1
      puts "    #{num}. #{item}"
    end
  end

  def strike(enemy_name, enemy_hp, seed)
    dmg = @attack + GameUtils.pseudo_rand(seed) % 6
    is_crit = seed.even?
    if is_crit
      dmg = dmg * CRIT_MULT
      puts "  CRITICAL HIT!"
    end
    remaining = GameUtils.clamp(enemy_hp - dmg, 0, 999)
    puts "  #{@name} attacks #{enemy_name} for #{dmg}! Enemy HP: #{remaining}"
    remaining
  end

  def take_damage(dmg)
    @hp = GameUtils.clamp(@hp - dmg, 0, MAX_HP)
    puts "  #{@name} takes #{dmg} damage! HP: #{@hp}"
  end

  def to_s
    "#{@name} [HP:#{@hp} ATK:#{@attack} Lv:#{@level}]"
  end
end

# -- Method with yield ----------------------------------------
def with_banner(label)
  puts ""
  puts "--- #{label} ---"
  yield
  puts ""
end

# ============================================================
#  MAIN QUEST
# ============================================================

GameUtils.banner("Quest for the Golden Hash")
puts ""

# Party setup
party = []
party << Hero.new("Aldric", MAX_HP, 12, "warrior")
party << Hero.new("Elara", 85, 18, "mage")
party << Hero.new("Vex", 90, 15, "rogue")

puts "Your party:"
party.each { |h| puts "  #{h.title}" }
puts ""
puts "Heroes created: #{Hero.count}"

# Warm up type inference for combat methods
warmup = party.first
warmup.strike("dummy", 0, 1)
warmup.take_damage(0)
warmup.hp = MAX_HP

# Monster bestiary as hash
bestiary = {
  "Goblin" => [30, 5, 25],
  "Troll" => [60, 10, 50],
  "Dragon" => [120, 20, 200]
}

stats_display = bestiary.transform_values { |v| "HP:#{v.first} ATK:#{v[1]}" }
puts ""
puts "Bestiary:"
stats_display.each do |(name, desc)|
  puts "  #{name.ljust(10)} #{desc}"
end

# Encounter loop
encounters = ["Goblin", "Goblin", "Troll", "Dragon"]
loot_table = {
  "Goblin" => Loot.new("Goblin Ear", 2),
  "Troll" => Loot.new("Troll Hide", 15),
  "Dragon" => Loot.new("Dragon Scale", 100)
}

encounter_idx = 0
while encounter_idx < encounters.length
  monster_name = encounters[encounter_idx]
  $turn += 1
  stats = bestiary.fetch(monster_name)
  enemy_hp = stats[0]
  enemy_atk = stats[1]
  xp_reward = stats[2]

  enc_num = encounter_idx + 1
  puts ""
  puts "--- Encounter #{enc_num}: A wild #{monster_name} appears! ---"

  hero = party[encounter_idx % party.count]
  seed = $turn * 7 + encounter_idx * 13

  # Combat loop
  while hero.alive? && enemy_hp > 0
    seed = GameUtils.pseudo_rand(seed)
    enemy_hp = hero.strike(monster_name, enemy_hp, seed)

    if enemy_hp > 0
      dmg = enemy_atk + GameUtils.pseudo_rand(seed) % 4
      puts "  #{monster_name} claws at #{hero.name} for #{dmg}!"
      hero.take_damage(dmg)
    end
  end

  if hero.alive?
    puts "  #{hero.name} defeats the #{monster_name}!"
    hero.gain_xp(xp_reward)

    # Loot drop
    if loot_table.has_key?(monster_name)
      hero.loot(loot_table.fetch(monster_name))
    end

    # Special dragon loot
    hero.loot(Loot.new("Golden Hash", 500)) if monster_name == "Dragon"
  else
    puts "  #{hero.name} has fallen!"
  end

  puts ""
  encounter_idx += 1
end

# ============================================================
#  POST-QUEST SUMMARY
# ============================================================

with_banner("QUEST COMPLETE") do
  GameUtils.banner("Results")

  # Sort by level, rank heroes
  ranked = party.sort_by { |h| h.level }.reverse
  puts ""
  puts "Final party rankings:"
  ranked.each_with_index do |hero, i|
    medal = i == 0 ? "GOLD" : (i == 1 ? "SILVER" : "BRONZE")
    puts "  #{medal}: #{hero}"
    hero.show_inventory
  end

  # Aggregate stats with &:symbol
  levels = party.map(&:level)
  total_xp = party.map(&:xp).sum
  all_alive = party.all? { |h| h.alive? }
  any_leveled = party.any? { |h| h.level > 1 }
  max_level = levels.max
  min_level = levels.min

  puts ""
  puts "Party stats:"
  puts "  Total XP earned: #{total_xp}"
  puts "  Level range: #{min_level}-#{max_level}"
  puts "  All survived: #{all_alive}"
  puts "  Any leveled up: #{any_leveled}"

  # String operations
  victory_msg = "victory"
  bangs = "! " * 5
  puts ""
  puts "  #{bangs}#{victory_msg.upcase.reverse}#{bangs}"

  # Float operations
  avg_hp = party.map { |h| h.hp }.sum.to_f / party.count.to_f
  puts "  Average HP: #{avg_hp.round(1)}"

  # Range operations
  xp_table = (1..max_level).map { |lv| lv * 100 }
  puts "  XP table: #{xp_table.join(", ")}"
  puts "  Total XP needed: #{xp_table.sum}"

  # Hash aggregation with each_with_object
  name_map = party.map { |h| h.title }.each_with_object({}) do |(title, acc)|
    first_word = title.split(" ").first
    acc[first_word] = title
  end
  puts ""
  puts "Heroes by name:"
  name_map.each do |(k, v)|
    puts "  #{k}: #{v}"
  end
end

# Exception handling
with_banner("BONUS: Exception Handling") do
  begin
    raise "The dungeon collapses!"
  rescue => e
    puts "  Caught: #{e}"
    puts "  But our heroes escape!"
  ensure
    puts "  (The adventure continues...)"
  end
end

# Safe navigation, nil coalescing, ternary, string ops
with_banner("EPILOGUE") do
  hero = party.first
  backup_name = hero&.name || "Unknown"
  puts "  Lead hero: #{backup_name}"

  # Ternary
  epilogue = $turn > 3 ? "epic" : "modest"
  puts "  After #{$turn} turns, it was an #{epilogue} adventure."

  # String methods
  title = "the end"
  puts "  #{title.upcase.center(30, "~")}"
  puts "  Reversed: #{title.reverse}"
  puts "  Starts with 'the': #{title.start_with?("the")}"
  puts "  Includes 'end': #{title.include?("end")}"
  puts "  Chars: #{title.chars.join("-")}"
  puts "  Length: #{title.length}"

  # Regex
  test = "Level 42 quest"
  if test =~ /Level (\d+)/
    puts "  Matched level pattern!"
  end

  # Integer methods
  puts "  Turn even: #{$turn.even?}"
  puts "  Turn odd: #{$turn.odd?}"

  # Select, reject, count on arrays
  strong = party.select { |h| h.attack > 15 }
  weak = party.reject { |h| h.attack > 15 }
  puts "  Strong heroes: #{strong.count}"
  puts "  Other heroes: #{weak.count}"

  # reduce
  total_atk = party.reduce(0) { |sum, h| sum + h.attack }
  puts "  Combined ATK: #{total_atk}"

  puts ""
  puts "  " + "FIN".center(30, "=")
end
