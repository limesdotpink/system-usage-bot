import "dotenv/config";
import { createRestAPIClient } from "masto";
import { OSUtils } from "node-os-utils";

const masto = createRestAPIClient({
  url: process.env.INSTANCE_URL,
  accessToken: process.env.BOT_ACCESS_TOKEN,
});

function usageBar(perc) {
  const filledN = Math.round((perc / 100) * 16);
  return `${"▰".repeat(filledN)}${"▱".repeat(16 - filledN)} ${perc.toFixed(1)}%`;
}

function humanReadableTime(t) {
  const up = new Date(t);
  const upDays = up.getUTCDate() - 1;
  const upHours = up.getUTCHours();
  const upMinutes = up.getUTCMinutes();
  const upSeconds = up.getUTCSeconds();

  let h = "";
  if (upDays) {
    h += `${upDays} day${upDays !== 1 ? "s" : ""}, `;
  }
  if (upHours || upDays > 0) {
    h += `${upHours} hour${upHours !== 1 ? "s" : ""}, `;
  }
  if (upMinutes || upHours + upDays > 0) {
    h += `${upMinutes} minute${upMinutes !== 1 ? "s" : ""} and `;
  }
  h += `${upSeconds} second${upSeconds !== 1 ? "s" : ""}`;

  return h;
}

async function toot() {
  const osutils = new OSUtils();
  const overview = await osutils.overview();

  const cpuAvg = overview.system.loadAverage.load5;
  const memSumm = overview.memory;
  const { data: diskSumm } = await osutils.disk.spaceOverview();
  const uptime = humanReadableTime(overview.system.uptime);

  const tootArray = [];

  tootArray.push(`CPU\n${usageBar(cpuAvg)}`);

  tootArray.push(
    `RAM (${memSumm.used} / ${memSumm.total})\n${usageBar(memSumm.usagePercentage)}`,
  );

  tootArray.push(
    `Disk (${diskSumm.used.toString()} / ${diskSumm.total.toString()})\n${usageBar(diskSumm.usagePercentage)}`,
  );

  tootArray.push(`Uptime\n${uptime}`);

  const status = await masto.v1.statuses.create({
    status: tootArray.join("\n\n"),
    visibility: "public",
  });

  console.log(`Tooted on ${new Date()}! ${status?.url}`);
}

toot();
