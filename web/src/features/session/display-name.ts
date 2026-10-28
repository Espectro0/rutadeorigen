import type { Actor } from "./types";

export function shortName(actor: Actor): string {
    const first = actor.firstName.trim();
    const initial = Array.from(actor.lastName.trim())[0] ?? "";

    if (first === "") {
        return actor.name !== "" ? actor.name : actor.email;
    }
    return initial === "" ? first : `${first} ${initial.toUpperCase()}.`;
}