// Port of weerstation/scripts/astrocalc.js (sun-position math) to a pure ESM module.
//
// Source: Proceedings of the Open source GIS - GRASS users conference 2002 - Trento, Italy
// "The solar radiation model for Open source GIS: implementation and applications"
// Jaroslav Hofierka, Marcel Uri
//
// Ported functions (line numbers refer to weerstation/scripts/astrocalc.js):
//   - sun_declination(doy)   -> sunDeclination(doy)      (astrocalc.js:252-257)
//   - timeequation(doy)      -> timeEquation(doy)         (astrocalc.js:265-270)
//   - timedifference(dec)    -> timeDifference(latRAD,dec)(astrocalc.js:259-263)
//   - sunrise(doy)/sunset(doy) -> sunTimes() below         (astrocalc.js:272-282)
//   - azimuthhight() sun-elevation term (h0)                (astrocalc.js:375-393)
//   - conv_to_time() local-clock conversion via SR/SS - longitude/15 - time_zone + summertime
//     (weerstation/scripts/pvdiary.js:810)
//
// Simplification (per task-10 brief): the original calc_cs2()/azimuthhight() clear-sky
// model (weerstation/scripts/pvdiary.js:987-1025) folds in per-string PV config (azimuth,
// roof pitch, temperature/panel/generator efficiency curves) that is not part of this
// module's interface. This port keeps only the sun-elevation geometry and models relative
// clear-sky irradiance as clamp(sin(h0), 0, ...) scaled to pvWatt -- it reproduces the
// "peaks at midday, zero at night" shape without the roof-angle/beam/diffuse/reflection
// breakdown, which is out of scope here (Task 11 or later can reintroduce PV-string specifics).
//
// DST approach: Belgium's EU DST rule is hardcoded (CET/UTC+1 standard, CEST/UTC+1+1 from
// the last Sunday of March 01:00 UTC through the last Sunday of October 01:00 UTC) -- this
// is the simplest approach that is correct for all years using the current EU rule, and
// matches the original code's implicit assumption of a fixed Belgium timezone.

const PI = Math.PI
const RAD = PI / 180
// height of sun at sunrise: solar radius + refraction (astrocalc.js:14)
const H_SUNRISE = -(50.0 / 60.0) * RAD
// Belgium standard time is UTC+1 (CET)
const STANDARD_OFFSET_HOURS = 1

function parseDate(date) {
  return {
    year: parseInt(date.slice(0, 4), 10),
    month: parseInt(date.slice(4, 6), 10),
    day: parseInt(date.slice(6, 8), 10),
  }
}

// astrocalc.js:234-244 (dayofyear), simplified to a pure day-of-year calculation
function dayOfYear(date) {
  const { year, month, day } = parseDate(date)
  const d1 = Date.UTC(year, 0, 1)
  const d2 = Date.UTC(year, month - 1, day)
  return Math.round((d2 - d1) / 86400000) + 1
}

// astrocalc.js:252-257
function sunDeclination(doy) {
  return 0.409526325277017 * Math.sin(0.0169060504029192 * (doy - 80.0856919827619))
}

// astrocalc.js:265-270
function timeEquation(doy) {
  return (
    -0.170869921174742 * Math.sin(0.0336997028793971 * doy + 0.465419984181394) -
    0.129890681040717 * Math.sin(0.0178674832556871 * doy - 0.167936777524864)
  )
}

// astrocalc.js:259-263 (declination and latitude passed in instead of globals)
function timeDifference(latRAD, declination) {
  return (
    (12.0 *
      Math.acos(
        (Math.sin(H_SUNRISE) - Math.sin(latRAD) * Math.sin(declination)) /
          (Math.cos(latRAD) * Math.cos(declination))
      )) /
    PI
  )
}

function lastSundayOfMonth(year, month) {
  // day 0 of the *next* month == last day of `month` (month is 1-indexed)
  const lastDay = new Date(Date.UTC(year, month, 0))
  return lastDay.getUTCDate() - lastDay.getUTCDay()
}

// EU rule: DST from last Sunday of March 01:00 UTC to last Sunday of October 01:00 UTC
function isDstBrussels(date) {
  const { year, month, day } = parseDate(date)
  const instant = Date.UTC(year, month - 1, day, 12) // noon on the given date
  const marchTransition = Date.UTC(year, 2, lastSundayOfMonth(year, 3), 1)
  const octTransition = Date.UTC(year, 9, lastSundayOfMonth(year, 10), 1)
  return instant >= marchTransition && instant < octTransition
}

// astrocalc.js:375-393 (azimuthhight), keeping only the sun-elevation (h0) term.
// `curr_time` is in "standard time" hours (no DST offset), matching the original's
// convention where curr_time excludes the summertime adjustment (see pvdiary.js:765-768,
// where `h - sta` strips the DST label back out before calling the astro formula).
function sunElevation(curr_time, doy, latRAD, lon) {
  const TE = timeEquation(doy)
  const DC = sunDeclination(doy)
  const cosdec = Math.cos(DC)
  const sindec = Math.sin(DC)
  const timeZone = -STANDARD_OFFSET_HOURS // JS-style sign convention used in astrocalc.js:47
  const timeSinceNoon = curr_time + lon / 15 + timeZone - 12 + TE
  const lha = timeSinceNoon * (1.0027379 - 1 / 365.25) * 15 * RAD
  const coslha = Math.cos(lha)
  const coslat = Math.cos(latRAD)
  const sinlat = Math.sin(latRAD)
  return Math.asin(sindec * sinlat + cosdec * coslha * coslat)
}

function formatHHMM(hoursDecimal) {
  const totalMinutes = ((Math.round(hoursDecimal * 60) % 1440) + 1440) % 1440
  const hh = Math.floor(totalMinutes / 60)
  const mm = totalMinutes % 60
  return `${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}`
}

// sunTimes(date, lat, lon) -> { sunrise: 'HH:MM', sunset: 'HH:MM' } local Brussels time
export function sunTimes(date, lat, lon) {
  const doy = dayOfYear(date)
  const latRAD = lat * RAD
  const DC = sunDeclination(doy)
  const TE = timeEquation(doy)
  const td = timeDifference(latRAD, DC)
  const SR = 12 - td - TE // astrocalc.js:272-276
  const SS = 12 + td - TE // astrocalc.js:278-282
  const dst = isDstBrussels(date) ? 1 : 0
  // local clock conversion, per pvdiary.js:810: SR - longitude/15 - time_zone + summertime
  const localSunrise = SR - lon / 15 + STANDARD_OFFSET_HOURS + dst
  const localSunset = SS - lon / 15 + STANDARD_OFFSET_HOURS + dst
  return { sunrise: formatHHMM(localSunrise), sunset: formatHHMM(localSunset) }
}

// clearSky(date, lat, lon, pvWatt) -> [{ t: 'HH:MM', w }] 5-min grid, 04:00-22:00
export function clearSky(date, lat, lon, pvWatt) {
  const doy = dayOfYear(date)
  const latRAD = lat * RAD
  const dst = isDstBrussels(date) ? 1 : 0
  const out = []
  for (let minutes = 4 * 60; minutes <= 22 * 60; minutes += 5) {
    const localHour = minutes / 60
    const stdHour = localHour - dst // strip DST label, see sunElevation() comment
    const h0 = sunElevation(stdHour, doy, latRAD, lon)
    const rel = h0 > 0 ? Math.sin(h0) : 0
    out.push({ t: formatHHMM(localHour), w: Math.round(rel * pvWatt) })
  }
  return out
}

// ---------------------------------------------------------------------------
// FULL per-string clear-sky model (faithful port of the legacy irradiance model)
//
// Ported from weerstation/scripts/astrocalc.js. Every function below reproduces
// its legacy counterpart term-for-term; the legacy module-level globals become
// members of a per-string `ctx` object (built by initAstro), so there is no
// module-level mutable state.
//
//   init_astrocalc()            (astrocalc.js:8-62)   -> initAstro()
//   global_radiation()          (astrocalc.js:303-305)-> folded into initAstro (G0)
//   fLTFarr()                   (astrocalc.js:116-148)-> nearestCity()
//   conv_avrg()                 (astrocalc.js:150-171)-> convAvrg()
//   direct_beam()               (astrocalc.js:307-317)-> directBeam()
//   diffuse_radiation()         (astrocalc.js:319-330)-> diffuseRadiation()
//   diffrad_coeff()             (astrocalc.js:332-364)-> diffradCoeff()
//   ground_reflection()         (astrocalc.js:366-370)-> groundReflection()
//   angular_reflection()        (astrocalc.js:187-191)-> angularReflection()
//   refraction()                (astrocalc.js:296-300)-> refraction()
//   real_temp()                 (astrocalc.js:214-218)-> realTemp()
//   windspeed()                 (astrocalc.js:194-197)-> windspeed()
//   panel_temp()                (astrocalc.js:221-226)-> panelTemp()
//   temperature_coeff()         (astrocalc.js:228-232)-> temperatureCoeff()
//   panel_coeff()               (astrocalc.js:181-185)-> panelCoeff()
//   generator_coeff()           (astrocalc.js:174-178)-> generatorCoeff()
//   azimuthhight() tot_en       (astrocalc.js:375-416)-> azimuthTotEn()
//   calc_cs2() per-string drive  (pvdiary.js:987-1025) -> clearSkyStrings()
//
// The driver (pvdiary.js:766-803) advances astro-time as `label - summertime`
// and computes power = round(tot_en/1000 * pv_watt * pv_eff) per string, summed
// over strings; pv_eff is 1 in the legacy code (astrocalc.js:45). Day energy is
// sum(power)/12 (12 five-minute samples per hour).

// astrocalc.js:47 -> time_zone = a.stdTimezoneOffset()/60 = -1 for Belgium (CET).
const TIME_ZONE = -STANDARD_OFFSET_HOURS

// astrocalc.js:17 meteo_data["Gemiddelde"] = [sunhours, avg-min-temp, avg-max-temp, windspeed]
const METEO = [
  [59, 77, 114, 159, 191, 188, 201, 190, 143, 113, 66, 45],
  [0.7, 0.7, 3.1, 5.3, 9.2, 11.9, 14.0, 13.6, 10.9, 7.8, 4.1, 1.6],
  [5.7, 6.6, 10.4, 14.2, 18.1, 20.6, 23, 22.6, 19, 14.7, 9.5, 6.1],
  [4.1, 3.8, 3.8, 3.4, 3.2, 3, 2.9, 2.8, 3, 3.4, 3.6, 3.7],
]

// astrocalc.js:116-135: Linke turbidity monthly tables per weather station.
// Each entry: [name, lat, lon, monthlyLTF[], icao]
const LTF_CITIES = [
  ['Ukkel', 50.801, 4.333, [3.3, 3.5, 4.1, 4.3, 4.4, 4.2, 4.4, 4.5, 4.1, 3.4, 3.4, 3.0]],
  ['Aartselaar', 51.134, 4.385, [3.2, 3.4, 4.0, 4.0, 4.3, 4.1, 4.3, 4.4, 4.0, 3.4, 3.3, 3.0]],
  ['Emblem', 51.162, 4.605, [3.3, 3.4, 4.0, 4.1, 4.3, 4.1, 4.3, 4.5, 4.1, 3.4, 3.4, 3.0]],
  ['Veurne', 51.073, 2.670, [3, 3.1, 3.8, 3.4, 3.9, 3.9, 4, 4.1, 3.7, 3, 3, 2.8]],
  ['Oostende', 51.217, 2.900, [3, 3, 3.8, 3.4, 3.9, 3.9, 4, 4.2, 3.7, 3, 3, 2.8]],
  ['Kortrijk', 50.823, 3.259, [3, 3, 3.9, 3.3, 3.8, 3.9, 3.8, 4.1, 3.5, 3, 3, 2.9]],
  ['Gent', 51.057, 3.720, [3.1, 3.1, 3.9, 3.7, 4.1, 4, 4.1, 4.3, 3.8, 3.2, 3.2, 2.9]],
  ['Turnhout', 51.322, 4.938, [3.2, 3.2, 3.9, 3.9, 4.2, 4.1, 4.3, 4.5, 4.1, 3.5, 3.2, 3.1]],
  ['Mechelen', 51.025, 4.477, [3.3, 3.5, 4.1, 4.1, 4.4, 4.1, 4.4, 4.5, 4.1, 3.4, 3.4, 3]],
  ['Leuven', 50.878, 4.704, [3.3, 3.5, 4.2, 4.3, 4.4, 4.2, 4.4, 4.6, 4.1, 3.5, 3.5, 3]],
  ['Hasselt', 50.931, 5.332, [3.3, 3.4, 4, 4.1, 4.4, 4.2, 4.5, 4.5, 4.1, 3.4, 3.4, 3]],
  ['Tienen', 50.808, 4.943, [3.3, 3.5, 4.1, 4.2, 4.4, 4.2, 4.4, 4.5, 4.1, 3.5, 3.4, 3]],
  ['Groningen', 53.2, 6.6, [2.6, 2.3, 3.1, 2.9, 3.2, 3.4, 3.9, 4.1, 3.9, 3.5, 2.6, 2.8, 3.2]],
  ['Amsterdam', 52.370, 4.895, [2.7, 2.5, 3.5, 3.2, 3.4, 3.8, 4, 4.5, 4, 3.6, 2.5, 2.7]],
  ['Rotterdam', 51.951, 4.434, [2.8, 2.6, 3.5, 3.3, 3.5, 3.8, 4, 4.5, 4, 3.6, 2.4, 2.8]],
  ['Arnhem', 52.068, 5.896, [2.7, 2.4, 3.5, 3.2, 3.5, 4, 4.3, 4.7, 4.1, 3.6, 2.5, 2.8]],
]

// pvdiary.js:1105-1109 daysInMonth(month, 2011) -> non-leap month lengths.
const DAYS_IN_MONTH = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]

// astrocalc.js:116-148: pick the LTF station nearest to the PV location (great-circle).
function nearestCity(lonRAD, latRAD) {
  const R = 6371
  let best = LTF_CITIES[0]
  let minD = Infinity
  for (const c of LTF_CITIES) {
    const lat2 = c[1] * RAD
    const lon2 = c[2] * RAD
    const d =
      Math.acos(
        Math.sin(latRAD) * Math.sin(lat2) +
          Math.cos(latRAD) * Math.cos(lat2) * Math.cos(lon2 - lonRAD)
      ) * R
    if (d < minD) {
      minD = d
      best = c
    }
  }
  return best
}

// astrocalc.js:150-171: monthly-table -> smooth daily value with a
// 1.5*m - (prev+next)/4 sharpening then linear interpolation between months.
function convAvrg(inArray, doy) {
  const arr = inArray.slice()
  let m = 0
  let d = 0
  while (d < doy - 15) {
    d += DAYS_IN_MONTH[m]
    m++
  }
  arr.unshift(arr[11])
  arr.push(arr[1])
  const out = []
  for (let i = 1; i <= 12; i++) {
    out[i] = 1.5 * arr[i] - (arr[i - 1] + arr[i + 1]) / 4
  }
  out[0] = out[12]
  out[13] = out[1]
  const basem = out[m]
  const nextm = out[m + 1]
  const ddiff = doy - d + 15
  return basem + ((nextm - basem) / 30) * ddiff
}

// astrocalc.js:296-300
function refraction(h0) {
  return (
    RAD * 0.061359 * (0.1594 + 1.123 * h0 + 0.065656 * h0 * h0) /
    (1 + 28.9344 * h0 + 277.3971 * h0 * h0)
  )
}

// astrocalc.js:307-317
function directBeam(h0ref, ctx) {
  if (h0ref < -1.5 * RAD) return 0
  const m = 1 / (Math.sin(h0ref) + 0.50572 * Math.pow(h0ref + 6.07995, -1.6364))
  let dR
  if (m <= 20)
    dR = 1 / (6.6296 + 1.7513 * m - 0.1202 * m ** 2 + 0.0065 * m ** 3 - 0.00013 * m ** 4)
  else dR = 1 / (10.4 + 0.718 * m)
  const a = ctx.G0 * Math.exp(-0.8662 * ctx.LTF * m * dR)
  return a > 0 ? a : 0
}

// astrocalc.js:319-330
function diffuseRadiation(h0, ctx) {
  const LTF = ctx.LTF
  const Tn = -0.015843 + 0.030543 * LTF + 0.0003797 * LTF ** 2
  let A1 = 0.26463 - 0.061581 * LTF + 0.0031408 * LTF ** 2
  if (Tn * A1 < 0.0022) A1 = 0.0022 / Tn
  const A2 = 2.0402 + 0.018945 * LTF - 0.011161 * LTF ** 2
  const A3 = -1.3025 + 0.039231 * LTF + 0.0085079 * LTF ** 2
  const Fd = A1 + A2 * Math.sin(h0) + A3 * Math.sin(h0) ** 2
  const a = ctx.G0 * Tn * Fd
  return a > 0 ? a : 0
}

// astrocalc.js:332-364
function diffradCoeff(B0c, Dhc, az, h0, beamCoeff, ctx) {
  const { skydome_coeff, pv_roof, pv_az, G0 } = ctx
  const Kb = Dhc / G0
  const gn = (skydome_coeff + Math.cos(pv_roof)) / 2
  let coeff
  if (beamCoeff > 0) {
    const N = 0.00333 - 0.415 * Kb - 0.6987 * Kb ** 2
    const FgN =
      gn + (Math.sin(pv_roof) - pv_roof * Math.cos(pv_roof) - PI * Math.sin(pv_roof / 2) ** 2) * N
    if (h0 >= 0.1) {
      coeff = FgN * (1 - Kb) + (Kb * beamCoeff) / Math.sin(h0)
    } else {
      let ALN = az - pv_az
      if (ALN > PI) ALN -= 2 * PI
      if (ALN < -PI) ALN += 2 * PI
      coeff = FgN * (1 - Kb) + (Kb * Math.sin(pv_roof) * Math.cos(ALN)) / (0.1 - 0.008 * h0)
    }
  } else {
    const N = 0.25227
    coeff =
      gn + (Math.sin(pv_roof) - pv_roof * Math.cos(pv_roof) - PI * Math.sin(pv_roof / 2) ** 2) * N
  }
  return coeff > 0 ? coeff : 0
}

// astrocalc.js:366-370
function groundReflection(B0c, Dhc, h0, ctx) {
  const gn = (ctx.skydome_coeff + 1) / 2
  const a = (0.2 * gn * (B0c * Math.sin(h0) + Dhc) * (1 - Math.cos(ctx.pv_roof))) / 2
  return a > 0 ? a : 0
}

// astrocalc.js:187-191
function angularReflection(angle) {
  const ar = 0.16
  return (1 - Math.exp((-1 * angle) / ar)) / (1 - Math.exp(-1 / ar))
}

// astrocalc.js:194-197
function windspeed(ctx) {
  const a = 1.05 - 0.05 * ctx.location_windspeed
  return a > 0 ? a : 0
}

// astrocalc.js:214-218
function realTemp(currTime, ctx) {
  return (
    ctx.location_min_temp_month +
    0.5 *
      (ctx.location_max_temp_month - ctx.location_min_temp_month) *
      (1 + Math.sin((PI * (currTime - ctx.SR)) / (ctx.SS - ctx.SR) - 0.5))
  )
}

// astrocalc.js:221-226 (noct=47)
function panelTemp(totEn, currTime, ctx) {
  const noct = 47
  return realTemp(currTime, ctx) + ((noct - 20) / 80) * 100 * totEn * windspeed(ctx)
}

// astrocalc.js:228-232
function temperatureCoeff(temp, ctx) {
  return 1 + ((temp - 25) * ctx.pv_temp_coeff) / 100
}

// astrocalc.js:181-185 (simulated panel efficiency curve)
function panelCoeff(x) {
  const a = -1.4 + 2.5 * Math.pow(x, 1 / 16) - 0.1 * x
  return a > 0 ? a : 0
}

// astrocalc.js:174-178 (simulated SMA SB5000-TL20 inverter curve)
function generatorCoeff(x) {
  const a = 0.07 + Math.pow(x, 1 / 14) - 0.1 * x
  return a > 0 ? a : 0
}

// astrocalc.js:8-62: build the per-string context (was init_astrocalc's globals).
function initAstro(date, lat, lon, azDeg, tiltDeg, tempCoeff) {
  const doy = dayOfYear(date)
  const latRAD = lat * RAD
  const DC = sunDeclination(doy)
  const TE = timeEquation(doy)
  const td = timeDifference(latRAD, DC)
  const SR = 12 - td - TE // astrocalc.js:272-276
  const SS = 12 + td - TE // astrocalc.js:278-282
  const G0 = 1367 * (1 + 0.03344 * Math.cos((2 * PI * doy) / 365.25 - 0.048869)) // astrocalc.js:303-305
  const city = nearestCity(lon * RAD, latRAD)
  return {
    doy,
    latRAD,
    longitude: lon,
    DC,
    TE,
    SR,
    SS,
    G0,
    LTF: convAvrg(city[3], doy),
    pv_az: azDeg * RAD,
    pv_roof: tiltDeg * RAD,
    pv_temp_coeff: tempCoeff,
    skydome_coeff: 1, // astrocalc.js:37 default when panels see the horizon
    location_min_temp_month: convAvrg(METEO[1], doy),
    location_max_temp_month: convAvrg(METEO[2], doy),
    location_windspeed: convAvrg(METEO[3], doy),
  }
}

// astrocalc.js:375-416: full tot_en (W/m2-equivalent, incl. temp/panel/inverter
// derating). currTime is standard (non-DST) decimal hours.
function azimuthTotEn(ctx, currTime) {
  const { DC, TE, longitude, latRAD, pv_roof, pv_az } = ctx
  const cosdec = Math.cos(DC)
  const sindec = Math.sin(DC)
  const timeSinceNoon = currTime + longitude / 15 + TIME_ZONE - 12 + TE
  const lha = timeSinceNoon * (1.0027379 - 1 / 365.25) * 15 * RAD
  const coslha = Math.cos(lha)
  const sinlha = Math.sin(lha)
  const coslat = Math.cos(latRAD)
  const sinlat = Math.sin(latRAD)
  let azimuth = Math.atan2(-cosdec * sinlha, sindec * coslat - cosdec * coslha * sinlat)
  if (azimuth < 0) azimuth += 2 * PI
  const h0 = Math.asin(sindec * sinlat + cosdec * coslha * coslat)
  const h0ref = h0 + refraction(h0)
  let beamCoeff =
    Math.cos(h0) * Math.sin(pv_roof) * Math.cos(pv_az - azimuth) + Math.sin(h0) * Math.cos(pv_roof)
  if (beamCoeff < 0) beamCoeff = 0
  let B0c = directBeam(h0ref, ctx)
  const Dhc = diffuseRadiation(h0, ctx)
  if (Dhc === 0) B0c = 0
  const gr = groundReflection(B0c, Dhc, h0, ctx)
  const DhcCoeff = diffradCoeff(B0c, Dhc, azimuth, h0, beamCoeff, ctx)
  const arCoeff = angularReflection(beamCoeff)
  let totEn = B0c * beamCoeff * arCoeff + Dhc * DhcCoeff + gr
  const tempPanel = panelTemp(totEn / 1000, currTime, ctx)
  const tempCoeff = temperatureCoeff(tempPanel, ctx)
  const panCoeff = panelCoeff((totEn / 1000) * tempCoeff)
  const genCoeff = generatorCoeff((totEn / 1000) * tempCoeff * panCoeff)
  return totEn * tempCoeff * panCoeff * genCoeff
}

// clearSkyStrings(date, lat, lon, strings) -> [{ t: 'HH:MM', w }] on the 5-min
// grid 04:00->22:00 (217 samples). w = summed AC power (W) over all strings.
// strings: [{ wp, az (compass deg), tilt (deg), temp_coeff }, ...]
export function clearSkyStrings(date, lat, lon, strings) {
  const dst = isDstBrussels(date) ? 1 : 0
  const ctxs = strings.map((s) => initAstro(date, lat, lon, s.az, s.tilt, s.temp_coeff))
  const out = []
  for (let minutes = 4 * 60; minutes <= 22 * 60; minutes += 5) {
    const localHour = minutes / 60
    const stdHour = localHour - dst // astro-time = label - summertime (pvdiary.js:768)
    let w = 0
    for (let i = 0; i < strings.length; i++) {
      // calc_cs2: power = round(tot_en/1000 * pv_watt * pv_eff), pv_eff = 1
      w += Math.round((azimuthTotEn(ctxs[i], stdHour) / 1000) * strings[i].wp)
    }
    out.push({ t: formatHHMM(localHour), w })
  }
  return out
}
