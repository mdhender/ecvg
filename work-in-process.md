# About work in process

Factories do not turn resources into finished units on the turn they are ordered.
Everything a factory group builds spends at least four turns as work in process (WIP) before it reaches the entity's storage.
This page explains how that pipeline behaves and why so many of the factory rules follow from it.

The pipeline described here is how manufacturing looks from the outside.
It is not a requirement for how the engine works inside, or for how the report shows WIP.

This page explains the proposed manufacturing pipeline.
None of it is built yet, and it is not authoritative: where it and [AGENTS.md](AGENTS.md) or the [unit tables](unit-tables.md) differ, they are right.

## A year in four turns

A year is 4 turns, and no unit can be built in less than a year.
WIP is how the game squares a quarterly turn with a yearly build time.

The capacity rule is stated by the year.
One FACT converts 20 MU times its TL of MTL and NMTL each year into units of the same total mass.
Factories build every manufactured unit: everything except population, cadres, FOOD, the resources (FUEL, GOLD, MTL, NMTL), and RP.
Mass is conserved: the [unit tables](unit-tables.md) give each unit's mass as the sum of its MTL and NMTL inputs.
A load is split between MTL and NMTL in the proportions of the unit's inputs.
So a FACT-1 building LFSU-2, at 6 MTL and 10 NMTL each, takes 20 MU in that same proportion, 7.5 MTL and 12.5 NMTL, and turns it into 1.25 LFSU-2 a year.
Small groups pay more than this for the same load ([Why small groups cost more](#why-small-groups-cost-more)).

## Each FACT carries its own load

The pipeline belongs to the individual FACT, not to the group.
A FACT takes on a load of 20 times its TL in MU, carries it through four steps, and only then takes on another.
Four rules shape that journey.

1. The pipeline takes at least four turns of work to complete.
2. The pipeline does not start until the FACT has the required MU of MTL and NMTL at the beginning of a turn.
3. Once a FACT claims its MTL and NMTL, it holds them until the pipeline is complete.
4. The pipeline does not advance unless the FACT has the fuel and labor it needs at the beginning of a turn.

The steps are numbered WIP-0 to WIP-4.
WIP-0 is a load that has been claimed but not yet worked; WIP-1, WIP-2, and WIP-3 are a quarter, a half, and three quarters complete; WIP-4 is finished.
On a report, the WIP that is 1/4, 1/2, and 3/4 complete is WIP-1, WIP-2, and WIP-3.

## A turn in the factory

Manufacturing is the last part of production: it comes after farming and mining, and before the dis-assembly and assembly phases.
The orders phase has already run by then, but the assemble orders only start their work in it, and the units are assembled in the assembly phase.
So a FACT assembled in turn N claims its first load in turn N+1.
Manufacturing runs in four stages.

### Resources

First, MTL and NMTL are handed out.
Factory groups are served in order of seniority, which is the order of their group numbers, lowest first.
Within a group, only FACT that are idle and hold no load are served, from the highest TL down.
A FACT that can not be given its full load, and the waste that goes with it, is skipped, and the distribution moves on to the next.

A group's WIP capacity is the total of its FACT's loads, 20 MU times the TL for each FACT.
The loads its FACT carry and its stockpile of WIP-4 share that capacity.
A FACT claims a load only if the whole load fits in what is left of that capacity.

The MTL and NMTL a FACT claims leave storage.
The load becomes its WIP-0, and the waste is lost.
The load keeps the mass and volume of the resources it was made from, so starting work neither frees nor uses space.

### Staff and power

Next, the groups are staffed and powered, in the same order.
Staff is labor: PRO and USK from the entity's worker cadre (WRKR), where assembled AUTO stand in for USK but never for PRO.
Power is FUEL.

Labor is allocated to a group before fuel.
Each FACT needs one PRO and three USK, each times the group's labor efficiency factor ([Why small groups cost more](#why-small-groups-cost-more)), and that is what the group needs to run at 100%; when there is less, the group runs at the share it can staff.
The group then claims fuel, 0.5 times the TL a turn, for every FACT that holds WIP and is staffed.
A FACT with no load claims none.

When staff or fuel is short, each goes in the same order: by group seniority, then by TL, and then to the most advanced work first, WIP-3, then WIP-2, WIP-1, and WIP-0.
Every FACT that ends this stage with a load, staff, and power works.

Staff is not assigned to a group for good.
The worker cadre is remade every turn in the resource allocation phase, and staff a group takes is never given back during the turn.

A FACT that lacks a load, staff, or power is idle for the rest of the turn.

### Processing

Every FACT that is not idle advances its load one step: WIP-0 becomes WIP-1, WIP-1 becomes WIP-2, WIP-2 becomes WIP-3, and WIP-3 becomes WIP-4.
A WIP-4 load passes at once from the FACT to its group's stockpile, and the FACT is free to claim a new load on the next turn.
Since the load already counted against the group's capacity, the stockpile always has room for it.
A group's loads and stockpile together never exceed its capacity, so no FACT is ever left holding WIP-3 it can not pass on.

### Output

Last, the group converts its stockpile of WIP-4 into as many whole lots of the unit it is producing as it can.
It converts no more than the entity has room for: a conversion that would take the entity over its capacity is not made, and the WIP-4 is held.
Whatever WIP-4 is left over stays in the group's stockpile for a later turn.
The units made are added to the entity's storage, unassembled.

## Following a group through the year

Take a group of 50,000 FACT-1 building LFSU-2, with resources, staff, and power to spare.
On its first turn of work every FACT claims 20 MU, and the group holds 1,000,000 MU of work.

| TURN OF WORK | WIP-1 (1/4)  | WIP-2 (1/2)  | WIP-3 (3/4)  | OUTPUT        |
|--------------|--------------|--------------|--------------|---------------|
| 1            | 1,000,000 MU |      0       |      0       |      0        |
| 2            |      0       | 1,000,000 MU |      0       |      0        |
| 3            |      0       |      0       | 1,000,000 MU |      0        |
| 4            |      0       |      0       |      0       | 62,500 LFSU-2 |

On the fourth turn the loads reach WIP-4, the group turns 1,000,000 MU into 62,500 LFSU-2, and every FACT is empty again.
On the fifth turn they all claim new loads and the cycle repeats.
The group does not finish a quarter of its year's output each turn, but takes it all a quarter of the way at a time.

A group whose FACT all start together delivers in a lump every fourth turn.
Its output is steady only when its FACT are spread across the steps.
That happens when FACT are added to the group on different turns, and it also happens when resources are short: the FACT that are skipped one turn start on a later one, and from then on they finish on a different turn from the rest.

## Why the load is claimed up front

Claiming a whole load before starting, and holding it, means a FACT never starts work it can not finish for want of materials.
A shortage of MTL or NMTL delays new work; it never strands work already under way.

Fuel and labor are different, because they are needed every turn.
When staff or fuel runs short, sending it to the most advanced work first means the loads closest to finishing are the last to stop.

The cost of this is that a FACT's materials are tied up for the whole year.
WIP is not directly accessible to the empire: it can not be traded, transferred, or used by another group.
The only way to get the materials back is to disassemble the FACT, which scraps its WIP ([Losing FACT](#losing-fact)).

## Stalls

A FACT that holds WIP but has not been given staff and power for the turn is stalled.
A stall loses nothing.
The FACT keeps its WIP exactly as it is and waits for staff and power to return, then carries on from the same step.
Stalls are why the pipeline takes at least four turns of work rather than exactly four turns.

A stall costs time, not materials.
Because the load was claimed up front, a FACT that stalls at WIP-2 needs nothing more than staff and power to finish; it never has to find its MTL and NMTL again.

## When the entity is full

A full entity backs the pipeline up from the far end.
The group stops converting WIP-4 it has no room to store, so the WIP-4 builds up in the stockpile.
The stockpile shares the group's capacity with the loads, so every MU it holds is an MU no FACT can claim.
Work already under way carries on and finishes into the stockpile, but empty FACT take on new loads only while there is room for them.
Once the stockpile fills the group's capacity, no FACT can claim a load until the entity has room for the group's output again.

Take a group of 4 FACT-1, with a WIP capacity of 80 MU, in an entity with no room for its output.
FACT A begins the turn empty, B, C, and D hold WIP-3, and the stockpile holds 10 MU of WIP-4.
The group holds 70 MU, so only 10 MU of its capacity is left, and A's load of 20 MU does not fit; A stays empty.
B, C, and D finish, and the stockpile grows to 70 MU.
On the next turn only 10 MU is left again, so none of the four can claim a load, and the group stands idle until the entity has room.

An idle FACT with no load claims no fuel, so a group stopped by a full entity burns no FUEL.
The load keeps the volume of its inputs, so WIP takes the same room in the entity at every step.
The space a full entity lacks is the room for the finished units.

## Why the group keeps a stockpile

A FACT's load is measured in mass, not in units, and the two rarely line up.
A FACT-1 makes 20 MU a year; a SENS-1 has a mass of 3,000 MU.
No single FACT-1 could ever finish a sensor.

Pooling finished loads at the group solves that.
The group makes whole lots from the stockpile, and a fraction of a lot waits for the next turn's loads.
For the same reason, the group as a whole sets the output, and every FACT in a group builds the same unit, the one the group is `producing`.

The stockpile is also the buffer between the factory and storage.
It lets work under way finish when the entity is full, and its share of the group's capacity is what stops new work.

## Why small groups cost more

Small factory groups are less efficient than large ones.
The proposed rules give a group two efficiency factors, one for labor and one for materials, both set by the number of FACT in the group.

| Number of FACT in Group  | Efficiency Factor (Labor) | Efficiency Factor (Materials) |
|--------------------------|---------------------------|-------------------------------|
|       1 ... 4            | 6                         | 3.00                          |
|       5 ... 49           | 5                         | 2.50                          |
|      50 ... 499          | 4                         | 2.00                          |
|     500 ... 4,999        | 3                         | 1.75                          |
|   5,000 ... 49,999       | 2                         | 1.50                          |
|  50,000 ... 199,999      | 1                         | 1.25                          |
| 200,000 ... infinity     | 0.5                       | 1.00                          |

The labor factor multiplies the PRO and USK each FACT needs.
The materials factor multiplies the MTL and NMTL a FACT must claim for its load.
Only the load itself goes into WIP; the rest is waste, removed from storage when the load is claimed.

A TRNS-1 takes 3 MTL and 1 NMTL, so it costs a group of 4 FACT, with a materials factor of 3, 9 MTL and 3 NMTL a unit.
The extra 6 MTL and 2 NMTL are waste.
A group of 200,000 FACT or more pays exactly the unit's inputs and wastes nothing.

The materials factor is never below 1, so mass is never created: the units made weigh what the load weighs, and a small group pays for its smallness in waste.

## Losing FACT

Nothing disassembles a group's units yet.
When an order does, its work will be done in the dis-assembly phase, which comes after production, so the FACT will have worked the turn before they come down.
A disassembled FACT can not take its WIP into storage, so its WIP is scrapped: converted back to MTL and NMTL in the unit's proportions, with 95% of them returned to storage, as the scrap order does.
Removing FACT also lowers the group's WIP capacity, and both its capacity and the room left in it are worked out again.
The FACT that remain keep their loads, which still fit, since each load is exactly its FACT's share of the capacity.
Any WIP-4 that would take the loads and stockpile together above the new capacity is excess, and it is scrapped too.
That keeps the group within its capacity, so its remaining FACT can always pass their WIP-3 on.
Most of the materials come back, but the work done on them does not, and nor does the waste paid when the load was claimed.
This is the only way an empire can recover materials held as WIP, and the only way to move the FACT that held them.

FACT destroyed in combat are treated the same way.
Combat is not built yet; it comes before resource allocation in the turn, so the destroyed FACT's WIP and any excess WIP-4 are scrapped, and the group's capacity worked out again, before the group manufactures.

## Combining groups

No order combines two factory groups yet.
If one is added, FACT own their WIP, so it leaves every load from WIP-0 to WIP-3 where it is: each FACT brings its load into the combined group.
The combined group owns both groups' stockpiles of WIP-4.

## Why retooling waits

No order changes what a factory group is `producing` yet.
When one does, the same picture says how it behaves.
A FACT that holds a load does not retool, even though the order is accepted; it retools itself once its load is finished.
A FACT with no load is free to retool at once.

## Research

Factories do not build research.
RP has no mass and is not manufactured from MTL and NMTL, so it would never fill a load or pass through the pipeline.
Instead, laboratory (LAB) units, assembled into laboratory groups, make it.
There are no laboratory groups yet.

A LAB is a unit of its own in the [unit tables](unit-tables.md), a little lighter than a FACT of the same TL.
It needs three PRO and one USK, where a FACT needs one PRO and three USK, and the same fuel, 0.5 times its TL a turn.
Each LAB produces 0.25 times its TL in RP for each turn it is allocated labor and fuel; a turn without them is simply a turn without research, with no WIP to stall or hold.
Laboratories produce just after factories in the turn, and their labor and fuel are allocated the same way: senior groups first, and within a group from the highest TL down.

Keeping research out of the factories keeps it out of the pipeline's rules altogether.
A FACT is never switched to or from research, so it never has to finish its WIP before research or retool after it.

## Farms and mines

Farms and mines are the other production units.
A MINE has no pipeline, and neither does a FARM-2 to FARM-10: they produce on the turn they work, with nothing held over.
A FARM-2 to FARM-10 makes 20 times its TL in FOOD a year, a quarter of it each turn it works, and a MINE extracts 25 times its TL times its deposit's yield each turn.
Their output needs no materials, so there is no load to claim and nothing to carry.

A FARM-1 is the exception.
It makes 100 FOOD a year, all at once, after 4 consecutive turns of work, and its group keeps a `pipeline` of the units with 1, 2, and 3 turns behind them.
Like a factory's, that pipeline goes to the units closest to delivering when staff or fuel is short.
Unlike a factory's, it does not stall: a FARM-1 that misses a turn loses its crop and starts over.

They draw on the same staff and power as factories.
Each FARM and MINE needs one PRO and three USK from the worker cadre, and both burn FUEL.
Resource allocation serves farming groups, then mining groups, then factory groups, so farms and mines have first claim on the entity's labor and fuel.
Staff is never given back during the turn, so a worker on a farm is not free to work in a mine or a factory later in the same turn.
A shortage falls on the factories before it falls on FOOD or the mines.

Within farming and within mining, labor and fuel are allocated by seniority, the lowest group number first.
A farming or mining group has a single TL, so there is no order of TL within it, as there is in a factory group.

The order also links them to the pipeline.
Mines produce before factories, so MTL and NMTL mined on a turn are in storage when the factories hand out loads that same turn.

Without WIP, a shortage of labor or fuel costs a mine, or a farm other than a FARM-1, only that turn's output.
There is no work to stall, and the next turn starts afresh.
Some of the units can produce while the rest stand idle; the ones that produce are those reached first in that order.

A lack of room is handled at the source rather than by a stockpile.
FOOD that does not fit is lost.
An open-air colony's mined product goes to its depot, which never runs short of room; what an enclosed colony has no room for is returned to the deposit.
A factory, by contrast, holds its finished work in its stockpile and stops taking new loads.

## Why seniority matters

Every allocation in production, of resources, staff, and fuel, serves the oldest group first.
Seniority is group number: an entity's farming, mining, and factory groups share one run of numbers, a new group always gets a higher number than any before it, and a number is never reused, so the lowest number is always the oldest group of its kind.
In a shortage the newest groups go without, and an empire can not renumber or reorder the groups.
The tool an empire has is the `set idle` order, which idles a percentage of a group's units; idled units ask for no staff or fuel, which leaves more for the rest.
