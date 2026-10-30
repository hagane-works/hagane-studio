import { describe, expect, it } from "vitest";
import type { GuiCountryStats } from "@/lib/guiRenderApi";
import {
  abbreviate,
  formatGameDate,
  fuelReadout,
  threatFrame,
  topbarBars,
  topbarValues,
} from "./topbarStats";

const CHINA: GuiCountryStats = {
  tag: "CHI",
  political_power: 0,
  command_power: 0,
  convoys: 40,
  stability: 0.2,
  war_support: 0.32,
  population: 326_975_088,
  manpower: 4_904_626,
  conscription: 0.015,
  civilian_factories: 16,
  military_factories: 11,
  dockyards: 2,
};

const GERMANY_1936: GuiCountryStats = {
  tag: "GER",
  political_power: 0,
  command_power: 0,
  convoys: 200,
  research_slots: 4,
  stability: 0.7,
  war_support: 0.3,
  manpower: 982_708,
  civilian_factories: 35,
  military_factories: 28,
  dockyards: 10,
  factories_total: 73,
  fuel: { days: 469, ratio: 0.8, capacity: 371_500, gaining: true },
  supply_ratio: 1,
  world_tension: 0,
  decisions_takeable: [{ id: "GER_cancel_mefos", name: "Cancel MEFO Bills" }],
  decisions_takeable_count: 1,
  faction: {
    name: "Axis",
    icon: "GFX_faction_logo_axis_miniature",
    initiative: 0,
  },
  army_experience: 0,
  navy_experience: 0,
  air_experience: 0,
  nukes: 0,
};

describe("topbarValues", () => {
  it("prints the country's own numbers the way the bar does", () => {
    const values = topbarValues(CHINA, "1936.1.1.12");
    expect(values.stability).toBe("20%");
    expect(values.war_support).toBe("32%");
    expect(values.manpower).toBe("4.90M");

    expect(values.industrial_capacity).toBe("29");
    expect(values.convoys_count).toBe("40");
    expect(values.DateText).toBe("12:00, 1 Jan, 1936");
  });

  it("takes the service's factory total when it sends one", () => {
    expect(topbarValues(GERMANY_1936, null).industrial_capacity).toBe("73");
  });

  it("prints the start values of command power and experience, whole", () => {
    const values = topbarValues(GERMANY_1936, "1936.1.1.12");
    expect(values.command_power_value).toBe("0");
    expect(values.experience_value).toBe("0");
    expect(values.navy_experience_value).toBe("0");
    expect(values.air_experience_value).toBe("0");

    const blitz = topbarValues(
      {
        ...GERMANY_1936,
        command_power: 130.4,
        army_experience: 55,
        air_experience: 25,
      },
      null,
    );
    expect(blitz.command_power_value).toBe("130");
    expect(blitz.experience_value).toBe("55");
    expect(blitz.air_experience_value).toBe("25");
  });

  it("floors political power, which the service now sends as a float", () => {
    expect(
      topbarValues({ tag: "X", political_power: 1077.8 }, null).pol_power,
    ).toBe("1077");
  });

  it("fills the run-time boxes with the files' start values", () => {
    const values = topbarValues(GERMANY_1936, "1936.1.1.12");
    expect(values.supply_value).toBe("100%");
    expect(values.threat_value).toBe("0%");
    expect(
      topbarValues({ ...GERMANY_1936, world_tension: 42 }, null).threat_value,
    ).toBe("42%");

    expect(values.decisionview_amount_to_take_items).toBe("1");

    expect(values.initiative_text).toBe("0");
  });

  it("leaves what an older service does not know to the .gui, and the fuel text to the layer", () => {
    const values = topbarValues(CHINA, "1936.1.1.12");

    expect(values.fuel_value).toBeUndefined();
    expect(values.supply_value).toBeUndefined();
    expect(values.threat_value).toBeUndefined();
    expect(values.decisionview_amount_to_take_items).toBeUndefined();

    expect(values.initiative_text).toBe("");
  });

  it("reads zero as zero rather than falling back to a placeholder", () => {
    expect(topbarValues({ tag: "X", political_power: 0 }, "").pol_power).toBe(
      "0",
    );
  });

  it("uses the localised month names it is given", () => {
    const months = [
      "1月",
      "2月",
      "3月",
      "4月",
      "5月",
      "6月",
      "7月",
      "8月",
      "9月",
      "10月",
      "11月",
      "12月",
    ];
    expect(topbarValues(CHINA, "1939.8.14.12", months).DateText).toBe(
      "12:00, 14 8月, 1939",
    );
  });
});

describe("topbarBars", () => {
  it("fills the fuel bar to the history's stock, not a flat start ratio", () => {
    const bars = topbarBars(GERMANY_1936);
    expect(bars.fuel_ratio_bar).toBe(0.8);
    expect(bars.supply_ratio_bar).toBe(1);
    expect(bars.convoys_ratio_bar).toBe(1);
    expect(bars.industrial_ratio_bar).toBe(1);
  });

  it("leaves the fuel bar empty when the service sends no fuel", () => {
    expect(topbarBars(CHINA).fuel_ratio_bar).toBe(0);
  });
});

describe("fuelReadout", () => {
  it("prints the days to fill in green with the rising icon while the stock grows", () => {
    expect(
      fuelReadout(GERMANY_1936, { DAY_UNIT_SHORT: "d", YEAR_UNIT_SHORT: "y" }),
    ).toEqual({ text: "1.3 y", color: "G", frame: 2 });

    const soon = { ...GERMANY_1936, fuel: { ...GERMANY_1936.fuel!, days: 59 } };
    expect(
      fuelReadout(soon, { DAY_UNIT_SHORT: "d", YEAR_UNIT_SHORT: "y" }),
    ).toEqual({ text: "59 d", color: "G", frame: 2 });
  });

  it("prints the short full form for a full stock", () => {
    const japan = {
      ...GERMANY_1936,
      fuel: { days: null, ratio: 1, capacity: 50_000, gaining: true },
    };
    expect(fuelReadout(japan, { FUEL_STOCKPILE_MAX_SHORT: "Full" })).toEqual({
      text: "Full",
      color: "G",
      frame: 1,
    });
  });

  it("says nothing for an older service", () => {
    expect(fuelReadout(CHINA)).toBeNull();
  });
});

describe("threatFrame", () => {
  it("picks the globe by tension: calm at 0, burning at 42", () => {
    expect(threatFrame(0)).toBe(1);
    expect(threatFrame(42)).toBe(5);
    expect(threatFrame(100)).toBe(10);
  });
});

describe("abbreviate", () => {
  it("matches the game's short form for big counts", () => {
    expect(abbreviate(4_904_626)).toBe("4.90M");
    expect(abbreviate(12_345)).toBe("12.35K");
    expect(abbreviate(940)).toBe("940");
  });
});

describe("formatGameDate", () => {
  it("prints the clock as the game does", () => {
    expect(formatGameDate("1939.8.14.12")).toBe("12:00, 14 Aug, 1939");
  });

  it("gives back anything it cannot read, rather than a wrong date", () => {
    expect(formatGameDate("not a date")).toBe("not a date");
    expect(formatGameDate(undefined)).toBe("");
  });
});
