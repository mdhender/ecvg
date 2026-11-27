# About work in process

Factories do not turn resources into finished units on the turn they are ordered.
Everything a factory group builds spends at least four turns as work in process (WIP) before it reaches the entity's storage ([H3.0](../rules/manual.md#rule-H3.0), [H3.4](../rules/manual.md#rule-H3.4)).
This page explains how that pipeline behaves and why so many of the factory rules follow from it.

The pipeline described here is how manufacturing looks from the outside.
It is not a requirement for how the engine works inside, or for how the report shows WIP.

This page explains the proposed manufacturing pipeline.
None of it is built yet, and it is not authoritative: where it and the [manual](../rules/manual.md) differ, the manual is right.
Rule numbers such as [H3.4](../rules/manual.md#rule-H3.4) link to the manual.
The manual calls an entity an S/C (ship or colony); its ID number is the entity ID.

## A year in four turns

A year is 4 turns ([A5.0](../rules/manual.md#rule-A5.0)), and no unit can be built in less than a year ([H3.0](../rules/manual.md#rule-H3.0)).
WIP is how the game squares a quarterly turn with a yearly build time.

The capacity rule is stated by the year.
One FACT converts 20 MU times its TL of MTL and NMTL each year into units of the same total mass ([H3.3](../rules/manual.md#rule-H3.3)).
Factories build everything except population, FOOD, and resources ([H3.0](../rules/manual.md#rule-H3.0)).
Mass is conserved: the unit characteristics give each unit's mass as the sum of its MTL and NMTL ([R3.0](../rules/manual.md#rule-R3.0)).
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

Manufacturing happens in the production stage, after farms and mines and before people are paid ([B3.17](../rules/manual.md#rule-B3.17)).
Every order has been processed by then, including the assemble orders of the assembly stage ([B3.8](../rules/manual.md#rule-B3.8)), so a group assembled this turn claims its first loads this turn.
It runs in four stages.

### Resources

First, MTL and NMTL are handed out.
Factory groups are served in order of seniority, which is the order of their group numbers, lowest first.
Within a group, only FACT that are idle and hold no load are served.
A FACT that can not be given its full load, and the waste that goes with it, is skipped, and the distribution moves on to the next.

A group's WIP capacity is the total of its FACT's loads, 20 MU times the TL for each FACT.
The loads its FACT carry and its stockpile of WIP-4 share that capacity.
A FACT claims a load only if the whole load fits in what is left of that capacity.

The MTL and NMTL a FACT claims leave storage.
The load becomes its WIP-0, and the waste is lost.
The load keeps the mass and volume of the resources it was made from, so starting work neither frees nor uses space.

### Staff and power

Next, the groups are staffed and powered, in the same order.
Staff is labor: PRO and USK from the entity's worker cadre (WRKR), where assembled AUTO stand in for USK but never for PRO ([H3.1](../rules/manual.md#rule-H3.1), [H3.10](../rules/manual.md#rule-H3.10)).
Power is FUEL.

Labor is allocated to a group before fuel.
Each FACT needs one PRO and three USK, each times the group's labor efficiency factor ([Why small groups cost more](#why-small-groups-cost-more)), and that is what the group needs to run at 100%; when there is less, the group runs at the share it can staff.
The group then claims fuel, 0.5 times the TL a turn ([H3.1](../rules/manual.md#rule-H3.1)), for every FACT that holds WIP and is staffed.
A FACT with no load claims none.

When staff or fuel is short, each goes in the same order: by group seniority, and then to the most advanced work first, WIP-3, then WIP-2, WIP-1, and WIP-0 ([H3.9](../rules/manual.md#rule-H3.9)).
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

Last, the group converts its stockpile of WIP-4 into as many of the units it is producing as it can.
It converts no more than the entity has room for: a conversion that would take the entity over its capacity ([F3.2.3](../rules/manual.md#rule-F3.2.3)) is not made, and the WIP-4 is held.
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
This is the pattern in the first example in [H3.4](../rules/manual.md#rule-H3.4): the group does not finish a quarter of its year's output each turn, but takes it all a quarter of the way at a time.

A group whose FACT all start together delivers in a lump every fourth turn.
Its output is steady only when its FACT are spread across the steps.
That happens when FACT are added to the group on different turns, as in the second example in [H3.4](../rules/manual.md#rule-H3.4), and it also happens when resources are short: the FACT that are skipped one turn start on a later one, and from then on they finish on a different turn from the rest.

## Why the load is claimed up front

Claiming a whole load before starting, and holding it, means a FACT never starts work it can not finish for want of materials.
A shortage of MTL or NMTL delays new work; it never strands work already under way.

Fuel and labor are different, because they are needed every turn.
When staff or fuel runs short, sending it to the most advanced work first means the loads closest to finishing are the last to stop.

The cost of this is that a FACT's materials are tied up for the whole year.
WIP is not directly accessible to the empire: it can not be traded, transferred, or used by another group.
The only way to get the materials back is to disassemble the FACT, which junks its WIP ([Losing FACT](#losing-fact)).

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
The group makes whole units from the stockpile, and a fraction of a unit waits for the next turn's loads.
For the same reason, the group as a whole sets the output, and every FACT in a group builds the same unit, the one the group is `producing` ([H3.0](../rules/manual.md#rule-H3.0)).

The stockpile is also the buffer between the factory and storage.
It lets work under way finish when the entity is full, and its share of the group's capacity is what stops new work.

## Why small groups cost more

Small factory groups are less efficient than large ones ([H3.2](../rules/manual.md#rule-H3.2)).
The manual has one efficiency factor; the proposed rules give a group two, one for labor and one for materials, both set by the number of FACT in the group.

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

Disassemble orders are processed in the dis-assembly stage, before production ([B3.4.1](../rules/manual.md#rule-B3.4.1)).
A disassembled FACT can not take its WIP into storage, so its WIP is junked: converted back to MTL and NMTL in the unit's proportions, with 90% of them returned to storage, the same loss as disassembly itself ([F10.3](../rules/manual.md#rule-F10.3)).
Removing FACT also lowers the group's WIP capacity, and both its capacity and the room left in it are worked out again.
The FACT that remain keep their loads, which still fit, since each load is exactly its FACT's share of the capacity.
Any WIP-4 that would take the loads and stockpile together above the new capacity is excess, and it is junked too.
That keeps the group within its capacity, so its remaining FACT can always pass their WIP-3 on.
Most of the materials come back, but the work done on them does not, and nor does the waste paid when the load was claimed.
This is the only way an empire can recover materials held as WIP, and the only way to move the FACT that held them.

FACT destroyed in combat are treated the same way.
Combat comes first in the turn ([B3.1](../rules/manual.md#rule-B3.1)), so the destroyed FACT's WIP and any excess WIP-4 are junked, and the group's capacity worked out again, before the group manufactures.

## Combining groups

FACT own their WIP, so a combine order (COM, [S8.2](../rules/manual.md#rule-S8.2)) leaves every load from WIP-0 to WIP-3 where it is: each FACT brings its load into the combined group.
The combined group owns both groups' stockpiles of WIP-4.
Combine orders are processed in the dis-assembly stage, before production ([B3.4.6](../rules/manual.md#rule-B3.4.6)), so the stockpiles combined are those left at the end of the previous turn's manufacturing.

## Why retooling waits

The retooling rules in [H3.6](../rules/manual.md#rule-H3.6) follow from the same picture.
A FACT that holds a load does not retool, even though the retool order is accepted; it retools itself once its load is finished.

The formula for how many FACT retool on a turn is the pipeline written as arithmetic.
Take the mass of everything in WIP as if it were complete and divide by 20 times the FACT's TL.
Since each FACT carries a load of 20 MU times its TL, the quotient is the number of FACT that still hold one.
Subtract it from the size of the group, and what is left is the FACT that are free to retool.

## Research

Factory groups also build research ([H3.7](../rules/manual.md#rule-H3.7)), and it does not go through the pipeline.
RSCH has no mass ([R3.0](../rules/manual.md#rule-R3.0)) and needs no MTL or NMTL ([J4.1](../rules/manual.md#rule-J4.1)), so it never fills a load.
A FACT building RSCH makes 0.25 times its TL in RSCH each turn ([J4.1](../rules/manual.md#rule-J4.1)).

A group ordered to build research finishes its WIP first ([J4.1](../rules/manual.md#rule-J4.1)): a FACT that holds a load carries it through, and turns to research when it is free.
Switching to research needs no retooling, but switching from research back to a unit does ([H3.6](../rules/manual.md#rule-H3.6)).

Research is going to move out of the factories to laboratory (LAB) units.
Until it does, this is how it works.

## Farms and mines

Farms and mines are the other production units, and neither has a pipeline.
A FARM produces FOOD and a MINE extracts ore on the turn they work, with nothing held over: 25 FOOD per FARM-1 or 5 times the TL for higher farms ([H1.1](../rules/manual.md#rule-H1.1)), and 25 times the TL in ore per MINE ([H2.1](../rules/manual.md#rule-H2.1)).
Their output is stated per turn, where a factory's is stated per year, and it needs no materials, so there is no load to claim and nothing to carry.

They draw on the same staff and power as factories.
Each FARM and MINE needs one PRO and three USK ([H1.4](../rules/manual.md#rule-H1.4), [H2.3](../rules/manual.md#rule-H2.3)), AUTO can stand in for the USK ([H3.10](../rules/manual.md#rule-H3.10)), and both use fuel ([H1.5](../rules/manual.md#rule-H1.5), [H2.3](../rules/manual.md#rule-H2.3)).
Because they come first in the production stage, farms, then mines, then factories ([B3.17](../rules/manual.md#rule-B3.17)), farms and mines have first claim on the entity's labor and fuel.
Staff is not given back until the end of the turn, so a worker on a farm is not free to work in a mine or a factory later in the same turn.
A shortage falls on the factories before it falls on FOOD or ore.

Within farming and within mining, labor and fuel are allocated in the factory order: senior groups first.

The order also links them to the pipeline.
Mines produce before factories, so MTL and NMTL mined on a turn are in storage when the factories hand out loads that same turn.

Without WIP, a shortage of labor or fuel costs a farm or mine only that turn's output ([H1.2](../rules/manual.md#rule-H1.2), [H2.2](../rules/manual.md#rule-H2.2)).
There is no work to stall, and the next turn starts afresh.
Some of the units can produce while the rest stand idle; the ones that produce are those reached first in that order.

A lack of room is handled at the source rather than by a stockpile.
FOOD that does not fit is thrown out ([H1.2](../rules/manual.md#rule-H1.2)).
Mines extract only the ore whose refined GOLD or FUEL will fit, while MTL and NMTL are stored outside the colony walls and never run short of room ([H2.2](../rules/manual.md#rule-H2.2)).
A factory, by contrast, holds its finished work in its stockpile and stops taking new loads.

## Why seniority matters

Every allocation in production, of resources, staff, and fuel, serves the oldest group first.
Seniority is group number: an entity's groups of every kind share one run of numbers, a new group always gets a higher number than any before it, and a number is never reused, so the lowest number is always the oldest group.
In a shortage the newest groups go without, and an empire can not renumber or reorder the groups ([H3.9](../rules/manual.md#rule-H3.9)).
An empire shuts down a group's units, all of them or a portion, with the shut down order ([S8.5](../rules/manual.md#rule-S8.5)); units that are shut down ask for no staff or fuel, which leaves more for the rest.
