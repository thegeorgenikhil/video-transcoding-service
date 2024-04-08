const video = document.querySelector("video");
const qualityPicker = document.getElementById("quality-picker");
let videos = {};

const GET_VIDEO_INFO_LAMBDA_URL =
  "";
const ACCESS_TOKEN = "";
const BUCKET_LINK = ""

window.onload = async () => {
  const url = new window.URLSearchParams(window.location.search).get("video");
  const res = await fetch(GET_VIDEO_INFO_LAMBDA_URL, {
    method: "POST",
    body: JSON.stringify({ access_token: ACCESS_TOKEN, video_key: url }),
  });
  const data = await res.json();
  videos = data.video.transcoding_files;
  const source = video.querySelector("source");
  source.src = BUCKET_LINK + videos["1080p"];
  video.load();
  video.play();
};

qualityPicker.addEventListener("change", changeVideoRes);
async function changeVideoRes(e) {
  const selectedValue = e.target.value;
  const source = video.querySelector("source");
  const updatedSrc = BUCKET_LINK + videos[selectedValue];

  const currTime = video.currentTime;

  if (updatedSrc !== source.src) {
    source.label = selectedValue;
    source.src = updatedSrc;
    video.load();
    video.currentTime = currTime;
    video.play();
  }
}
