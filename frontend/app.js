const videoInput = document.getElementById("videoInput");
const uploadBtn = document.getElementById("uploadBtn");
const outputDiv = document.getElementById("outputDiv");
const videoListDiv = document.getElementById("videoListDiv");
const toastEl = document.getElementById("snackbar");
let setTimeoutId;

const UPLOAD_LAMBDA_URL =
  "";
const ACCESS_TOKEN = "";

window.onload = async () => {
  showToast(toastEl, "Loading videos...", "green");
  const videos = await getVideos();
  renderVideoList(videos);
};

async function getVideos() {
  try {
    const res = await fetch(
      "",
      {
        method: "POST",
        body: JSON.stringify({ access_token: ACCESS_TOKEN }),
      }
    );
    const data = await res.json();
    if (!res.ok) {
      throw new Error("Error: getting upload URL, " + data.message);
    }

    return data.videos;
  } catch (error) {
    console.error(error);
    showToast(toastEl, error.message, "red");
  }
}

function renderVideoList(videos) {
  // Clear existing content
  videoListDiv.innerHTML = "";

  // Create table element
  const table = document.createElement("table");
  table.classList.add("video-table");

  // Create table header row
  const headerRow = table.insertRow();
  const headers = [
    { name: "Video Name", class: "" },
    { name: "Status", class: "mobile-hidden" },
    { name: "Time Taken", class: "mobile-hidden" },
    { name: "Uploaded At", class: "mobile-hidden" },
    { name: "Play Video", class: "" },
  ];
  headers.forEach((header) => {
    const th = document.createElement("th");
    th.textContent = header.name;
    if (header.class) {
      th.classList.add(header.class);
    }
    headerRow.appendChild(th);
  });

  // Iterate through videos and create table rows
  videos.forEach((video) => {
    const row = table.insertRow();

    // Video Key
    const keyCell = row.insertCell();
    keyCell.textContent = truncateMiddle(video["key"], 20);

    // Status
    const statusCell = row.insertCell();
    statusCell.classList.add("mobile-hidden");
    statusCell.textContent = video["status"];

    // Transcoding Time
    const transcodingTimeCell = row.insertCell();
    transcodingTimeCell.classList.add("mobile-hidden");
    if (video["transcoding_time"]) {
      transcodingTimeCell.textContent = formattedSecondsToMinutes(
        video["transcoding_time"]
      );
    }

    // Uploaded At
    const uploadedAtCell = row.insertCell();
    uploadedAtCell.classList.add("mobile-hidden");
    uploadedAtCell.textContent = new Date(
      video["uploaded_at"]
    ).toLocaleString();

    const playVideoCell = row.insertCell();
    if (video["status"] != "completed") {
      playVideoCell.classList.add("col-disabled");
    }
    playVideoCell.innerHTML = `<a href="/frontend/play.html?video=${video.key}">
    <img src="./icons/play.png" style="width: 20px;height: 20px" />
    </a>`;
    playVideoCell.ariaDisabled = true;
  });

  // Append table to the videoListDiv
  videoListDiv.appendChild(table);
}

uploadBtn.addEventListener("click", async () => {
  try {
    showToast(toastEl, "Uploading video...");
    const file = videoInput.files[0];
    if (!file) {
      showToast(toastEl, "Please select a video to upload", "red");
      return;
    }
    const fileName = file.name;

    const res = await fetch(UPLOAD_LAMBDA_URL, {
      method: "POST",
      body: JSON.stringify({ file_name: fileName, access_token: ACCESS_TOKEN }),
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error("Error: getting upload URL, " + data.message);
    }

    const videoUploadRes = await fetch(data.upload_url, {
      method: "PUT",
      body: file,
      headers: {
        "Content-Type": file.type,
      },
    });
    if (!videoUploadRes.ok) {
      throw new Error("Error: while upload video to S3, " + data.message);
    }
    videoInput.value = "";
  } catch (error) {
    console.error(error);
    showToast(toastEl, error.message, "red");
  }
});

function showToast(toastEl, message, color = "#0d6efd") {
  toastEl.className = "";

  clearTimeout(setTimeoutId);

  // Add the "show" class to DIV
  toastEl.className = "show";
  toastEl.textContent = message;
  toastEl.style.backgroundColor = color;
  // After 2 seconds, remove the show class from DIV
  setTimeoutId = setTimeout(function () {
    toastEl.className = "";
  }, 2000);
}

function truncateMiddle(str, maxLength) {
  if (str.length > maxLength) {
    const start = str.substring(0, maxLength / 2);
    const end = str.substring(str.length - maxLength / 2);
    return start + "..." + end;
  }
  return str;
}

function formattedSecondsToMinutes(seconds) {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes > 0) {
    return `${minutes}m ${remainingSeconds.toFixed(0)}s`;
  }

  return `${remainingSeconds.toFixed(0)}s`;
}
