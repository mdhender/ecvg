# Empyrean Challenge - Unit Table

**Status**: Draft

## Units and Classes

| Item Code | Item Name           | Class          |
|-----------|---------------------|----------------|
| AMSL      | Anti-Missile        | Ammunition     |
| ACFT      | Assault Craft       | Weaponry       |
| AWPN      | Assault Weapons     | Weaponry       |
| AUTO      | Automation          | Infrastructure |
| CNGD      | Consumer Goods      | Commodity      |
| CSUP      | Combat Supplies     | Ammunition     |
| CWRK      | Construction Worker | Cadre          |
| ESHD      | Energy Shield       | Weaponry       |
| EWPN      | Energy Weapon       | Weaponry       |
| FACT      | Factory             | Infrastructure |
| FARM      | Farm                | Infrastructure |
| FOOD      | Food                | Commodity      |
| FUEL      | Fuel                | Resource       |
| HDRV      | Hyper Drive         | Propulsion     |
| LAB       | Laboratory          | Infrastructure |
| LFSU      | Life Support        | Static         |
| METL      | Metallics           | Resource       |
| MRBT      | Military Robot      | Weaponry       |
| MINE      | Mine                | Infrastructure |
| MLNC      | Missile Launcher    | Weaponry       |
| MSSL      | Missile             | Ammunition     |
| NMTL      | Non-Metallics       | Resource       |
| POL       | Police              | Cadre          |
| POWR      | Power Plant         | Infrastructure |
| PRO       | Professionals       | Living         |
| PRTO      | Prototype           | Technology     |
| PROB      | Probe               | Recon          |
| RBL       | Rebel               | Cadre          |
| RP        | Research Point      | Technology     |
| SENS      | Sensor              | Recon          |
| SLD       | Soldiers            | Living         |
| SDRV      | Space Drive         | Propulsion     |
| SPAG      | Special Agent       | Cadre          |
| SPY       | Spy                 | Cadre          |
| STRC      | Structure           | Structural     |
| STRL      | Light Structure     | Structural     |
| TRNE      | Trainee             | Cadre          |
| TRNS      | Transport           | Transportation |
| UEM       | Unemployables       | Living         |
| USK       | Unskilled Workers   | Living         |
| WRKR      | Worker              | Cadre          |

## Lot Sizes

A lot is the number of units a single build adds to inventory.
The **Unit Inputs** and **Unit Mass and Volume** tables give values for a whole lot.
Units not listed here have a lot size of 1, so their values are also per unit.

| Item Code | Lot Size  |
|-----------|-----------|
| CNGD      | 10        |
| CSUP      | 50        |
| PROB      | 10 x TL^2 |
| STRC      | 10        |
| STRL      | 100       |

Inventory counts individual units, not lots.
The mass or volume of a quantity is `quantity x lot value / lot size`, computed on the total and rounded up once.
40 PROB-3 have a mass of 223 MU (222.2 rounded up).
The general rule for rounding is that it always goes against the player:
costs, mass, and volume round up; mine and farm output rounds down.
Combat values round against the attacker and in favor of the defender.
Manufactured units are built in whole lots.

## Unit Inputs

> **NOTE**
> Units that are not manufactured (Resources, Food, Living, Cadres, and Research Points) are excluded from this table.

| Item Code | Metals to Build | Non-Metals to Build |
|-----------|-----------------|---------------------|
| AMSL      | 2 x TL          | 2 x TL              |
| ACFT      | 3 x TL          | 2 x TL              |
| AWPN      | 1 x TL          | 1 x TL              |
| AUTO      | 2 x TL          | 2 x TL              |
| CNGD      | 2               | 4                   |
| CSUP      | 1               | 1                   |
| ESHD      | 10 x TL         | 10 x TL             |
| EWPN      | 5 x TL          | 5 x TL              |
| FACT      | TL + 8          | 4 + TL              |
| FARM      | TL + 4          | 2 + TL              |
| HDRV      | 25 x TL         | 20 x TL             |
| LAB       | TL + 5          | TL + 3              |
| LFSU      | 3 x TL          | 5 x TL              |
| MRBT      | TL + 10         | TL + 10             |
| MINE      | TL + 5          | TL + 5              |
| MLNC      | 15 x TL         | 10 x TL             |
| MSSL      | 2 x TL          | 2 x TL              |
| POWR      | TL + 5          | TL + 5              |
| PRTO      | TBD             | TBD                 |
| PROB      | 200             | 300                 |
| SENS      | 1000 x TL       | 2000 x TL           |
| SDRV      | 15 x TL         | 10 x TL             |
| STRC      | 7 x TL          | 3 x TL              |
| STRL      | 7 x TL          | 3 x TL              |
| TRNS      | 3 x TL          | 1 x TL              |

## Units Requiring Assembly

These units must be assembled before they can be used.

| Item Code | ASM? |
|-----------|------|
| AUTO      | yes  |
| ESHD      | yes  |
| EWPN      | yes  |
| FACT      | yes  |
| FARM      | yes  |
| HDRV      | yes  |
| LAB       | yes  |
| LFSU      | yes  |
| MINE      | yes  |
| MLNC      | yes  |
| POWR      | yes  |
| SENS      | yes  |
| SDRV      | yes  |
| STRC      | yes  |
| STRL      | yes  |

## Unit Mass and Volume

| Item Code | Mass            | Volume          | Volume Stowed   |
|-----------|-----------------|-----------------|-----------------|
| AMSL      | 4 x TL          | 4 x TL          | 2 x TL          |
| ACFT      | 5 x TL          | 5 x TL          | 2.5 x TL        |
| AWPN      | 2 x TL          | 2 x TL          | 1 x TL          |
| AUTO      | 4 x TL          | 4 x TL          | 2 x TL          |
| CNGD      | 6               | 3               | 3               |
| CSUP      | 2               | 2               | 2               |
| ESHD      | 20 x TL         | 20 x TL         | 10 x TL         |
| EWPN      | 10 x TL         | 10 x TL         | 5 x TL          |
| FACT      | 2 x (TL + 6)    | 2 x (TL + 6)    | TL + 6          |
| FARM      | 2 x (TL + 3)    | 2 x (TL + 3)    | TL + 3          |
| FOOD      | 6               | 3               | 3               |
| FUEL      | 1               | 0.5             | 0.5             |
| HDRV      | 45 x TL         | 45 x TL         | 22.5 x TL       |
| LAB       | 2 x TL + 8      | 2 x TL + 8      | TL + 4          |
| LFSU      | 8 x TL          | 8 x TL          | 4 x TL          |
| METL      | 1               | 0.5             | 0.5             |
| MRBT      | 2 x (TL + 10)   | 2 x (TL + 10)   | TL + 10         |
| MINE      | 2 x (TL + 5)    | 2 x (TL + 5)    | TL + 5          |
| MLNC      | 25 x TL         | 25 x TL         | 12.5 x TL       |
| MSSL      | 4 x TL          | 4 x TL          | 4 x TL          |
| NMTL      | 1               | 0.5             | 0.5             |
| POWR      | 2 x (TL + 5)    | 2 x (TL + 5)    | TL + 5          |
| PRTO      | TL              | 3 x TL          | 3 x TL          |
| PROB      | 500             | 500             | 500             |
| RP        | 0               | 0               | 0               |
| SENS      | 3000 x TL       | 3000 x TL       | 1500 x TL       |
| SDRV      | 25 x TL         | 25 x TL         | 12.5 x TL       |
| STRC      | 10 x TL         | 10 x TL         | 10 x TL         |
| STRL      | 10 x TL         | 10 x TL         | 10 x TL         |
| TRNS      | 4 x TL          | 4 x TL          | 4 x TL          |

> **NOTE**
> All Population Units have Mass of 1 MU and Volume of 1 VU.
> Population Units can never be stowed.

> **NOTE**
> Stowed volume is rounded up after multiplying by the quantity, never per unit.
> 10 stowed ACFT-1 occupy 25 VU; 11 occupy 28 VU (27.5 rounded up).

## Unit Notes

| Item Code | Operational Requirements                                               | Output and Notes                                                                     |
|-----------|------------------------------------------------------------------------|--------------------------------------------------------------------------------------|
| AMSL      | Missile Launcher of same TL                                            | Destroys Missiles see combat                                                         |
| ACFT      | 1 soldier or military robot equivalent + 0.1 fuel in combat            | Provides 10 x TL^2 combat factors does not require transports to attack              |
| AWPN      | 1 soldier or military robot equivalent                                 | Provides 2 x TL^2 combat factors                                                     |
| AUTO      |                                                                        | Replaces 1 x TL^2 Unskilled see Automation in Production chapter                     |
| CWRK      |                                                                        | Provide manpower to assemble, disassemble, add on and scrap items to/from SCs        |
| CNGD      |                                                                        | Consumption determines S.O.L.                                                        |
| CSUP      |                                                                        | Consumed in combat - 1 per soldier per combat round                                  |
| ESHD      | 1 soldier / 100 and uses 10 x TL fuel only                             | Deflects 10 x TL^2 energy units per use                                              |
| EWPN      | 1 soldier / 100 and uses 4 x TL fuel only                              | Destroys 10 x TL^2 mass per hit                                                      |
| FACT      | 1 professional 3 unskilled and 0.5 x TL fuel or power                  | Produces 20 x TL mass per turn see Manufacturing                                     |
| FARM      | 1 professional 3 unskilled fuel or power varies                        | Production varies via TL see Farming                                                 |
| FOOD      |                                                                        | Feeds 4 to 16 population each turn see Basic Needs                                   |
| FUEL      |                                                                        | Raw material used by many units                                                      |
| HDRV      | 1 professional / 100 and 40 fuel only per light year                   | Lift capacity 1045 x TL range is (Square root of TL) + 4                             |
| LAB       | 3 professional 1 unskilled, 0.5 x TL fuel or power                     | Produces 0.25 x TL research per turn                                                 |
| LFSU      | 1 x TL fuel or power                                                   | Supports 1 x TL^2 population                                                         |
| METL      |                                                                        | Raw material used in production                                                      |
| MRBT      |                                                                        | Same as 2 x TL soldiers                                                              |
| MINE      | 1 professional 3 unskilled, 0.5 x TL fuel or power                     | Mines 25 x TL per turn in raw ore. Actual net depends on yield of deposit see Mining |
| MLNC      | 1 soldier / 100                                                        | Launches 1 missile per attack see Combat                                             |
| MSSL      | Missile Launcher of same TL                                            | Destroys 100 x TL^2 Mass                                                             |
| NMTL      |                                                                        | Raw material used in production                                                      |
| POL       |                                                                        | Drafted to keep the peace and suppress rebellion by arresting rebels                 |
| POWR      | Open Surface Colonies ONLY                                             | Produces TL Power per turn (think hydro electric)                                    |
| PRO       | Food and Consumer Goods                                                | Provide administrative and managerial services to operate certain assembled items    |
| PRTO      |                                                                        | Transfer Tech Levels between Entities                                                |
| PROB      |                                                                        | One attempt per probe, then expended; built in lots of 10 x TL^2                      |
| RBL       |                                                                        | Tally of population willing to rebel; does not allocate the underlying units         |
| RP        |                                                                        | Expended to increase TLs                                                             |
| SENS      | 0.05 x TL fuel or power                                                | Used to obtain probe information                                                     |
| SLD       | Food and Consumer Goods as well as 1 x combat supplies during combat   | Provide combat factors and also staffing for certain weapons systems                 |
| SDRV      | 1 professional / 100 and 1 x TL fuel only                              | Produces 3000 x TL^2 thrust                                                          |
| SPAG      |                                                                        | Used to infiltrate rebel population sectors and assist police in locating rebels     |
| SPY       |                                                                        | Used to perform espionage                                                            |
| STRC      | Must be assembled to enclose volume                                    | Encloses (1 x TL^2) divided by structural requirement (based on Entity type)         |
| STRL      | Must be assembled to enclose volume                                    | Light structure; encloses (1 x TL^2) divided by structural requirement (based on Entity type) |
| TRNE      | Professionals to perform Training                                      | Trainees are in training to become Professionals                                     |
| TRNS      | 1 professional / 10 and 0.1 x TL^2 fuel                                | Transports 20 x TL^2 Mass; In Combat = 3 x TL^2 Mass                                 |
| UEM       | Food and Consumer Goods                                                | Population growth occurs in this category                                            |
| USK       | Food and Consumer Goods                                                | Provide labor to operate certain assembled items                                     |
| WRKR      | N/A                                                                    | PRO and USK units allocated to a FACT, FARM, LAB, or MINE                            |

## Enclosed Space

Assembled `STRC` and `STRL` units create space for their entity.
The space each unit creates is (1 x TL^2) divided by the entity's structural requirement.
Stowed structure creates no space and occupies its full volume.
Assembled structure does not count against the space it creates, but its mass still counts when moving the ship.

`STRL` creates the same space as `STRC` for a tenth of the materials and mass, with two limits:

- `STRL` can only be built in Enclosed Orbital Colonies.
- `STRC` absorbs more damage in combat.

| Entity Type             | Structural Requirement | Space per Unit |
|-------------------------|------------------------|----------------|
| Ship                    | 10                     | 0.1 x TL^2 VU  |
| Open Air Surface Colony | 1                      | 1 x TL^2 VU    |
| Enclosed Surface Colony | 5                      | 0.2 x TL^2 VU  |
| Enclosed Orbital Colony | 10                     | 0.1 x TL^2 VU  |

## Draft: Unit-Definition Catalog

> **Draft.**
> One row per valid `(unit_code, tech_level)` pair.
> This is one option for satisfying the domain invariants, not a required persistence design.
> Fixed-point encoding of the numeric columns is unsettled; values are shown as computed.

Generated from the tables above:
* `lot_size` from **Lot Sizes**,
* `metl_input` and `nmtl_input` from **Unit Inputs**,
* `mass_units` from **Mass**,
* `volume_units` from **Volume**,
* `stowed_volume_units` from **Volume Stowed** (all three in **Unit Mass and Volume**), and
* `requires_assembly?` from **Units Requiring Assembly** (`ASM?`).

The `tech_level?` flag marks whether a unit carries Tech Levels.

Generation rules applied:

- Resource units (`FUEL`, `METL`, `NMTL`) and `FOOD` have only a TL0 row.
- Living (`PRO`, `SLD`, `UEM`, `USK`) and Cadre (`CWRK`, `POL`, `RBL`, `SPAG`, `SPY`, `TRNE`, `WRKR`) entries have no rows.
- `RP` is a non-physical bookkeeping unit and has no row.
- All remaining units have TL1–TL10 rows, except `CNGD` and `CSUP` (TL0 only).
- Invariant: a unit code never has both a TL0 row and a positive-TL row.
- `stowed_volume_units` may be fractional (`ACFT`, `HDRV`, `MLNC`, `SDRV` at odd Tech Levels); see the rounding note under **Unit Mass and Volume**.
- `metl_input`, `nmtl_input`, `mass_units`, `volume_units`, and `stowed_volume_units` are per lot; divide by `lot_size` for a single unit.

Notes:

- Values derive from the consolidated (future-game) tables above and are subject to change.


| unit_code | tech_level? | requires_assembly? |
|-----------|-------------|--------------------|
| ACFT      | yes         | no                 |
| AMSL      | yes         | no                 |
| AUTO      | yes         | yes                |
| AWPN      | yes         | no                 |
| CNGD      | no          | no                 |
| CSUP      | no          | no                 |
| ESHD      | yes         | yes                |
| EWPN      | yes         | yes                |
| FACT      | yes         | yes                |
| FARM      | yes         | yes                |
| FOOD      | no          | no                 |
| FUEL      | no          | no                 |
| HDRV      | yes         | yes                |
| LAB       | yes         | yes                |
| LFSU      | yes         | yes                |
| METL      | no          | no                 |
| MINE      | yes         | yes                |
| MLNC      | yes         | yes                |
| MRBT      | yes         | no                 |
| MSSL      | yes         | no                 |
| NMTL      | no          | no                 |
| POWR      | yes         | yes                |
| PROB      | yes         | no                 |
| PRTO      | yes         | no                 |
| SDRV      | yes         | yes                |
| SENS      | yes         | yes                |
| STRC      | yes         | yes                |
| STRL      | yes         | yes                |
| TRNS      | yes         | no                 |


| unit_code | tech_level | lot_size | metl_input | nmtl_input | mass_units | volume_units | stowed_volume_units |
|-----------|------------|----------|------------|------------|------------|--------------|---------------------|
| ACFT      | 1          | 1        | 3          | 2          | 5          | 5            | 2.5                 |
| ACFT      | 2          | 1        | 6          | 4          | 10         | 10           | 5                   |
| ACFT      | 3          | 1        | 9          | 6          | 15         | 15           | 7.5                 |
| ACFT      | 4          | 1        | 12         | 8          | 20         | 20           | 10                  |
| ACFT      | 5          | 1        | 15         | 10         | 25         | 25           | 12.5                |
| ACFT      | 6          | 1        | 18         | 12         | 30         | 30           | 15                  |
| ACFT      | 7          | 1        | 21         | 14         | 35         | 35           | 17.5                |
| ACFT      | 8          | 1        | 24         | 16         | 40         | 40           | 20                  |
| ACFT      | 9          | 1        | 27         | 18         | 45         | 45           | 22.5                |
| ACFT      | 10         | 1        | 30         | 20         | 50         | 50           | 25                  |
| AMSL      | 1          | 1        | 2          | 2          | 4          | 4            | 2                   |
| AMSL      | 2          | 1        | 4          | 4          | 8          | 8            | 4                   |
| AMSL      | 3          | 1        | 6          | 6          | 12         | 12           | 6                   |
| AMSL      | 4          | 1        | 8          | 8          | 16         | 16           | 8                   |
| AMSL      | 5          | 1        | 10         | 10         | 20         | 20           | 10                  |
| AMSL      | 6          | 1        | 12         | 12         | 24         | 24           | 12                  |
| AMSL      | 7          | 1        | 14         | 14         | 28         | 28           | 14                  |
| AMSL      | 8          | 1        | 16         | 16         | 32         | 32           | 16                  |
| AMSL      | 9          | 1        | 18         | 18         | 36         | 36           | 18                  |
| AMSL      | 10         | 1        | 20         | 20         | 40         | 40           | 20                  |
| AUTO      | 1          | 1        | 2          | 2          | 4          | 4            | 2                   |
| AUTO      | 2          | 1        | 4          | 4          | 8          | 8            | 4                   |
| AUTO      | 3          | 1        | 6          | 6          | 12         | 12           | 6                   |
| AUTO      | 4          | 1        | 8          | 8          | 16         | 16           | 8                   |
| AUTO      | 5          | 1        | 10         | 10         | 20         | 20           | 10                  |
| AUTO      | 6          | 1        | 12         | 12         | 24         | 24           | 12                  |
| AUTO      | 7          | 1        | 14         | 14         | 28         | 28           | 14                  |
| AUTO      | 8          | 1        | 16         | 16         | 32         | 32           | 16                  |
| AUTO      | 9          | 1        | 18         | 18         | 36         | 36           | 18                  |
| AUTO      | 10         | 1        | 20         | 20         | 40         | 40           | 20                  |
| AWPN      | 1          | 1        | 1          | 1          | 2          | 2            | 1                   |
| AWPN      | 2          | 1        | 2          | 2          | 4          | 4            | 2                   |
| AWPN      | 3          | 1        | 3          | 3          | 6          | 6            | 3                   |
| AWPN      | 4          | 1        | 4          | 4          | 8          | 8            | 4                   |
| AWPN      | 5          | 1        | 5          | 5          | 10         | 10           | 5                   |
| AWPN      | 6          | 1        | 6          | 6          | 12         | 12           | 6                   |
| AWPN      | 7          | 1        | 7          | 7          | 14         | 14           | 7                   |
| AWPN      | 8          | 1        | 8          | 8          | 16         | 16           | 8                   |
| AWPN      | 9          | 1        | 9          | 9          | 18         | 18           | 9                   |
| AWPN      | 10         | 1        | 10         | 10         | 20         | 20           | 10                  |
| CNGD      | 0          | 10       | 2          | 4          | 6          | 3            | 3                   |
| CSUP      | 0          | 50       | 1          | 1          | 2          | 2            | 2                   |
| ESHD      | 1          | 1        | 10         | 10         | 20         | 20           | 10                  |
| ESHD      | 2          | 1        | 20         | 20         | 40         | 40           | 20                  |
| ESHD      | 3          | 1        | 30         | 30         | 60         | 60           | 30                  |
| ESHD      | 4          | 1        | 40         | 40         | 80         | 80           | 40                  |
| ESHD      | 5          | 1        | 50         | 50         | 100        | 100          | 50                  |
| ESHD      | 6          | 1        | 60         | 60         | 120        | 120          | 60                  |
| ESHD      | 7          | 1        | 70         | 70         | 140        | 140          | 70                  |
| ESHD      | 8          | 1        | 80         | 80         | 160        | 160          | 80                  |
| ESHD      | 9          | 1        | 90         | 90         | 180        | 180          | 90                  |
| ESHD      | 10         | 1        | 100        | 100        | 200        | 200          | 100                 |
| EWPN      | 1          | 1        | 5          | 5          | 10         | 10           | 5                   |
| EWPN      | 2          | 1        | 10         | 10         | 20         | 20           | 10                  |
| EWPN      | 3          | 1        | 15         | 15         | 30         | 30           | 15                  |
| EWPN      | 4          | 1        | 20         | 20         | 40         | 40           | 20                  |
| EWPN      | 5          | 1        | 25         | 25         | 50         | 50           | 25                  |
| EWPN      | 6          | 1        | 30         | 30         | 60         | 60           | 30                  |
| EWPN      | 7          | 1        | 35         | 35         | 70         | 70           | 35                  |
| EWPN      | 8          | 1        | 40         | 40         | 80         | 80           | 40                  |
| EWPN      | 9          | 1        | 45         | 45         | 90         | 90           | 45                  |
| EWPN      | 10         | 1        | 50         | 50         | 100        | 100          | 50                  |
| FACT      | 1          | 1        | 9          | 5          | 14         | 14           | 7                   |
| FACT      | 2          | 1        | 10         | 6          | 16         | 16           | 8                   |
| FACT      | 3          | 1        | 11         | 7          | 18         | 18           | 9                   |
| FACT      | 4          | 1        | 12         | 8          | 20         | 20           | 10                  |
| FACT      | 5          | 1        | 13         | 9          | 22         | 22           | 11                  |
| FACT      | 6          | 1        | 14         | 10         | 24         | 24           | 12                  |
| FACT      | 7          | 1        | 15         | 11         | 26         | 26           | 13                  |
| FACT      | 8          | 1        | 16         | 12         | 28         | 28           | 14                  |
| FACT      | 9          | 1        | 17         | 13         | 30         | 30           | 15                  |
| FACT      | 10         | 1        | 18         | 14         | 32         | 32           | 16                  |
| FARM      | 1          | 1        | 5          | 3          | 8          | 8            | 4                   |
| FARM      | 2          | 1        | 6          | 4          | 10         | 10           | 5                   |
| FARM      | 3          | 1        | 7          | 5          | 12         | 12           | 6                   |
| FARM      | 4          | 1        | 8          | 6          | 14         | 14           | 7                   |
| FARM      | 5          | 1        | 9          | 7          | 16         | 16           | 8                   |
| FARM      | 6          | 1        | 10         | 8          | 18         | 18           | 9                   |
| FARM      | 7          | 1        | 11         | 9          | 20         | 20           | 10                  |
| FARM      | 8          | 1        | 12         | 10         | 22         | 22           | 11                  |
| FARM      | 9          | 1        | 13         | 11         | 24         | 24           | 12                  |
| FARM      | 10         | 1        | 14         | 12         | 26         | 26           | 13                  |
| FOOD      | 0          | 1        | N/A        | N/A        | 6          | 3            | 3                   |
| FUEL      | 0          | 1        | N/A        | N/A        | 1          | 0.5          | 0.5                 |
| HDRV      | 1          | 1        | 25         | 20         | 45         | 45           | 22.5                |
| HDRV      | 2          | 1        | 50         | 40         | 90         | 90           | 45                  |
| HDRV      | 3          | 1        | 75         | 60         | 135        | 135          | 67.5                |
| HDRV      | 4          | 1        | 100        | 80         | 180        | 180          | 90                  |
| HDRV      | 5          | 1        | 125        | 100        | 225        | 225          | 112.5               |
| HDRV      | 6          | 1        | 150        | 120        | 270        | 270          | 135                 |
| HDRV      | 7          | 1        | 175        | 140        | 315        | 315          | 157.5               |
| HDRV      | 8          | 1        | 200        | 160        | 360        | 360          | 180                 |
| HDRV      | 9          | 1        | 225        | 180        | 405        | 405          | 202.5               |
| HDRV      | 10         | 1        | 250        | 200        | 450        | 450          | 225                 |
| LAB       | 1          | 1        | 6          | 4          | 10         | 10           | 5                   |
| LAB       | 2          | 1        | 7          | 5          | 12         | 12           | 6                   |
| LAB       | 3          | 1        | 8          | 6          | 14         | 14           | 7                   |
| LAB       | 4          | 1        | 9          | 7          | 16         | 16           | 8                   |
| LAB       | 5          | 1        | 10         | 8          | 18         | 18           | 9                   |
| LAB       | 6          | 1        | 11         | 9          | 20         | 20           | 10                  |
| LAB       | 7          | 1        | 12         | 10         | 22         | 22           | 11                  |
| LAB       | 8          | 1        | 13         | 11         | 24         | 24           | 12                  |
| LAB       | 9          | 1        | 14         | 12         | 26         | 26           | 13                  |
| LAB       | 10         | 1        | 15         | 13         | 28         | 28           | 14                  |
| LFSU      | 1          | 1        | 3          | 5          | 8          | 8            | 4                   |
| LFSU      | 2          | 1        | 6          | 10         | 16         | 16           | 8                   |
| LFSU      | 3          | 1        | 9          | 15         | 24         | 24           | 12                  |
| LFSU      | 4          | 1        | 12         | 20         | 32         | 32           | 16                  |
| LFSU      | 5          | 1        | 15         | 25         | 40         | 40           | 20                  |
| LFSU      | 6          | 1        | 18         | 30         | 48         | 48           | 24                  |
| LFSU      | 7          | 1        | 21         | 35         | 56         | 56           | 28                  |
| LFSU      | 8          | 1        | 24         | 40         | 64         | 64           | 32                  |
| LFSU      | 9          | 1        | 27         | 45         | 72         | 72           | 36                  |
| LFSU      | 10         | 1        | 30         | 50         | 80         | 80           | 40                  |
| METL      | 0          | 1        | N/A        | N/A        | 1          | 0.5          | 0.5                 |
| MINE      | 1          | 1        | 6          | 6          | 12         | 12           | 6                   |
| MINE      | 2          | 1        | 7          | 7          | 14         | 14           | 7                   |
| MINE      | 3          | 1        | 8          | 8          | 16         | 16           | 8                   |
| MINE      | 4          | 1        | 9          | 9          | 18         | 18           | 9                   |
| MINE      | 5          | 1        | 10         | 10         | 20         | 20           | 10                  |
| MINE      | 6          | 1        | 11         | 11         | 22         | 22           | 11                  |
| MINE      | 7          | 1        | 12         | 12         | 24         | 24           | 12                  |
| MINE      | 8          | 1        | 13         | 13         | 26         | 26           | 13                  |
| MINE      | 9          | 1        | 14         | 14         | 28         | 28           | 14                  |
| MINE      | 10         | 1        | 15         | 15         | 30         | 30           | 15                  |
| MLNC      | 1          | 1        | 15         | 10         | 25         | 25           | 12.5                |
| MLNC      | 2          | 1        | 30         | 20         | 50         | 50           | 25                  |
| MLNC      | 3          | 1        | 45         | 30         | 75         | 75           | 37.5                |
| MLNC      | 4          | 1        | 60         | 40         | 100        | 100          | 50                  |
| MLNC      | 5          | 1        | 75         | 50         | 125        | 125          | 62.5                |
| MLNC      | 6          | 1        | 90         | 60         | 150        | 150          | 75                  |
| MLNC      | 7          | 1        | 105        | 70         | 175        | 175          | 87.5                |
| MLNC      | 8          | 1        | 120        | 80         | 200        | 200          | 100                 |
| MLNC      | 9          | 1        | 135        | 90         | 225        | 225          | 112.5               |
| MLNC      | 10         | 1        | 150        | 100        | 250        | 250          | 125                 |
| MRBT      | 1          | 1        | 11         | 11         | 22         | 22           | 11                  |
| MRBT      | 2          | 1        | 12         | 12         | 24         | 24           | 12                  |
| MRBT      | 3          | 1        | 13         | 13         | 26         | 26           | 13                  |
| MRBT      | 4          | 1        | 14         | 14         | 28         | 28           | 14                  |
| MRBT      | 5          | 1        | 15         | 15         | 30         | 30           | 15                  |
| MRBT      | 6          | 1        | 16         | 16         | 32         | 32           | 16                  |
| MRBT      | 7          | 1        | 17         | 17         | 34         | 34           | 17                  |
| MRBT      | 8          | 1        | 18         | 18         | 36         | 36           | 18                  |
| MRBT      | 9          | 1        | 19         | 19         | 38         | 38           | 19                  |
| MRBT      | 10         | 1        | 20         | 20         | 40         | 40           | 20                  |
| MSSL      | 1          | 1        | 2          | 2          | 4          | 4            | 4                   |
| MSSL      | 2          | 1        | 4          | 4          | 8          | 8            | 8                   |
| MSSL      | 3          | 1        | 6          | 6          | 12         | 12           | 12                  |
| MSSL      | 4          | 1        | 8          | 8          | 16         | 16           | 16                  |
| MSSL      | 5          | 1        | 10         | 10         | 20         | 20           | 20                  |
| MSSL      | 6          | 1        | 12         | 12         | 24         | 24           | 24                  |
| MSSL      | 7          | 1        | 14         | 14         | 28         | 28           | 28                  |
| MSSL      | 8          | 1        | 16         | 16         | 32         | 32           | 32                  |
| MSSL      | 9          | 1        | 18         | 18         | 36         | 36           | 36                  |
| MSSL      | 10         | 1        | 20         | 20         | 40         | 40           | 40                  |
| NMTL      | 0          | 1        | N/A        | N/A        | 1          | 0.5          | 0.5                 |
| POWR      | 1          | 1        | 6          | 6          | 12         | 12           | 6                   |
| POWR      | 2          | 1        | 7          | 7          | 14         | 14           | 7                   |
| POWR      | 3          | 1        | 8          | 8          | 16         | 16           | 8                   |
| POWR      | 4          | 1        | 9          | 9          | 18         | 18           | 9                   |
| POWR      | 5          | 1        | 10         | 10         | 20         | 20           | 10                  |
| POWR      | 6          | 1        | 11         | 11         | 22         | 22           | 11                  |
| POWR      | 7          | 1        | 12         | 12         | 24         | 24           | 12                  |
| POWR      | 8          | 1        | 13         | 13         | 26         | 26           | 13                  |
| POWR      | 9          | 1        | 14         | 14         | 28         | 28           | 14                  |
| POWR      | 10         | 1        | 15         | 15         | 30         | 30           | 15                  |
| PROB      | 1          | 10       | 200        | 300        | 500        | 500          | 500                 |
| PROB      | 2          | 40       | 200        | 300        | 500        | 500          | 500                 |
| PROB      | 3          | 90       | 200        | 300        | 500        | 500          | 500                 |
| PROB      | 4          | 160      | 200        | 300        | 500        | 500          | 500                 |
| PROB      | 5          | 250      | 200        | 300        | 500        | 500          | 500                 |
| PROB      | 6          | 360      | 200        | 300        | 500        | 500          | 500                 |
| PROB      | 7          | 490      | 200        | 300        | 500        | 500          | 500                 |
| PROB      | 8          | 640      | 200        | 300        | 500        | 500          | 500                 |
| PROB      | 9          | 810      | 200        | 300        | 500        | 500          | 500                 |
| PROB      | 10         | 1000     | 200        | 300        | 500        | 500          | 500                 |
| PRTO      | 1          | 1        | TBD        | TBD        | 1          | 3            | 3                   |
| PRTO      | 2          | 1        | TBD        | TBD        | 2          | 6            | 6                   |
| PRTO      | 3          | 1        | TBD        | TBD        | 3          | 9            | 9                   |
| PRTO      | 4          | 1        | TBD        | TBD        | 4          | 12           | 12                  |
| PRTO      | 5          | 1        | TBD        | TBD        | 5          | 15           | 15                  |
| PRTO      | 6          | 1        | TBD        | TBD        | 6          | 18           | 18                  |
| PRTO      | 7          | 1        | TBD        | TBD        | 7          | 21           | 21                  |
| PRTO      | 8          | 1        | TBD        | TBD        | 8          | 24           | 24                  |
| PRTO      | 9          | 1        | TBD        | TBD        | 9          | 27           | 27                  |
| PRTO      | 10         | 1        | TBD        | TBD        | 10         | 30           | 30                  |
| SDRV      | 1          | 1        | 15         | 10         | 25         | 25           | 12.5                |
| SDRV      | 2          | 1        | 30         | 20         | 50         | 50           | 25                  |
| SDRV      | 3          | 1        | 45         | 30         | 75         | 75           | 37.5                |
| SDRV      | 4          | 1        | 60         | 40         | 100        | 100          | 50                  |
| SDRV      | 5          | 1        | 75         | 50         | 125        | 125          | 62.5                |
| SDRV      | 6          | 1        | 90         | 60         | 150        | 150          | 75                  |
| SDRV      | 7          | 1        | 105        | 70         | 175        | 175          | 87.5                |
| SDRV      | 8          | 1        | 120        | 80         | 200        | 200          | 100                 |
| SDRV      | 9          | 1        | 135        | 90         | 225        | 225          | 112.5               |
| SDRV      | 10         | 1        | 150        | 100        | 250        | 250          | 125                 |
| SENS      | 1          | 1        | 1000       | 2000       | 3000       | 3000         | 1500                |
| SENS      | 2          | 1        | 2000       | 4000       | 6000       | 6000         | 3000                |
| SENS      | 3          | 1        | 3000       | 6000       | 9000       | 9000         | 4500                |
| SENS      | 4          | 1        | 4000       | 8000       | 12000      | 12000        | 6000                |
| SENS      | 5          | 1        | 5000       | 10000      | 15000      | 15000        | 7500                |
| SENS      | 6          | 1        | 6000       | 12000      | 18000      | 18000        | 9000                |
| SENS      | 7          | 1        | 7000       | 14000      | 21000      | 21000        | 10500               |
| SENS      | 8          | 1        | 8000       | 16000      | 24000      | 24000        | 12000               |
| SENS      | 9          | 1        | 9000       | 18000      | 27000      | 27000        | 13500               |
| SENS      | 10         | 1        | 10000      | 20000      | 30000      | 30000        | 15000               |
| STRC      | 1          | 10       | 7          | 3          | 10         | 10           | 10                  |
| STRC      | 2          | 10       | 14         | 6          | 20         | 20           | 20                  |
| STRC      | 3          | 10       | 21         | 9          | 30         | 30           | 30                  |
| STRC      | 4          | 10       | 28         | 12         | 40         | 40           | 40                  |
| STRC      | 5          | 10       | 35         | 15         | 50         | 50           | 50                  |
| STRC      | 6          | 10       | 42         | 18         | 60         | 60           | 60                  |
| STRC      | 7          | 10       | 49         | 21         | 70         | 70           | 70                  |
| STRC      | 8          | 10       | 56         | 24         | 80         | 80           | 80                  |
| STRC      | 9          | 10       | 63         | 27         | 90         | 90           | 90                  |
| STRC      | 10         | 10       | 70         | 30         | 100        | 100          | 100                 |
| STRL      | 1          | 100      | 7          | 3          | 10         | 10           | 10                  |
| STRL      | 2          | 100      | 14         | 6          | 20         | 20           | 20                  |
| STRL      | 3          | 100      | 21         | 9          | 30         | 30           | 30                  |
| STRL      | 4          | 100      | 28         | 12         | 40         | 40           | 40                  |
| STRL      | 5          | 100      | 35         | 15         | 50         | 50           | 50                  |
| STRL      | 6          | 100      | 42         | 18         | 60         | 60           | 60                  |
| STRL      | 7          | 100      | 49         | 21         | 70         | 70           | 70                  |
| STRL      | 8          | 100      | 56         | 24         | 80         | 80           | 80                  |
| STRL      | 9          | 100      | 63         | 27         | 90         | 90           | 90                  |
| STRL      | 10         | 100      | 70         | 30         | 100        | 100          | 100                 |
| TRNS      | 1          | 1        | 3          | 1          | 4          | 4            | 4                   |
| TRNS      | 2          | 1        | 6          | 2          | 8          | 8            | 8                   |
| TRNS      | 3          | 1        | 9          | 3          | 12         | 12           | 12                  |
| TRNS      | 4          | 1        | 12         | 4          | 16         | 16           | 16                  |
| TRNS      | 5          | 1        | 15         | 5          | 20         | 20           | 20                  |
| TRNS      | 6          | 1        | 18         | 6          | 24         | 24           | 24                  |
| TRNS      | 7          | 1        | 21         | 7          | 28         | 28           | 28                  |
| TRNS      | 8          | 1        | 24         | 8          | 32         | 32           | 32                  |
| TRNS      | 9          | 1        | 27         | 9          | 36         | 36           | 36                  |
| TRNS      | 10         | 1        | 30         | 10         | 40         | 40           | 40                  |
