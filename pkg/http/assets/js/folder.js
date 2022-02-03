// thanks https://stackoverflow.com/a/20732091/1128596
function humanFileSize(size) {
    if (size == 0) return 0;
    var i = Math.floor(Math.log(size) / Math.log(1024));
    return (size / Math.pow(1024, i)).toFixed(2) * 1 + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
};

function alert(message, type) {
    var wrapper = document.createElement('div')
    wrapper.innerHTML = '<div class="alert alert-' + type + ' alert-dismissible" role="alert">' + message +
        '<button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button></div>'

    var alertPlaceholder = document.getElementById('liveAlertPlaceholder');
    alertPlaceholder.append(wrapper)
};

function fileInfo(file) {
    var jelm = $('#offcanvasRightLabel');
    jelm.text(file);
    var body = $('#fileinfoBody');
    body.empty();

    var download = $('#offcanvasDownloadLink');
    download.attr('href', document.location.protocol + '//' + document.location.host + '/api/download?filename=' +
        file);
    download.attr('download', document.location.protocol + '//' + document.location.host + '/api/download?filename=' +
        file);

    var offcanvas = new bootstrap.Offcanvas($('#offcanvasRight'));
    offcanvas.show();

    var add = function (title, text, name) {
        var element = $(
                '<div class="list-group-item justify-content-between text-break" style="font-family:monospace;"></div>')
            .text(text);
        var title = $('<div class="fw-bold"></div>').text(title);
        var info = $(`<a data-bs-toggle="tooltip" data-bs-placement="left" data-bs-original-title="Provided by ` +
            name + `"> </a>`);
        new bootstrap.Tooltip(info);
        // https://icons.getbootstrap.com/icons/info/
        var icon = $(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-info-circle" viewBox="0 0 16 16">
<path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16z"/>
<path d="m8.93 6.588-2.29.287-.082.38.45.083c.294.07.352.176.288.469l-.738 3.468c-.194.897.105 1.319.808 1.319.545 0 1.178-.252 1.465-.598l.088-.416c-.2.176-.492.246-.686.246-.275 0-.375-.193-.304-.533L8.93 6.588zM9 4.5a1 1 0 1 1-2 0 1 1 0 0 1 2 0z"/>
</svg>`);

        info.append(icon);
        title.append(info);
        element.prepend(title);
        body.append(element);
    };

    var socket = new WebSocket("ws://" + document.location.host + "/api/fileinfo?filename=" + file);
    socket.onmessage = function (event) {
        var json = JSON.parse(event.data);
        if (json.title != "") {
            add(json.title, json.human, json.name);
        } else {
            add(json.name, json.human, json.name);
        }
    };
};

$(document).ready(function () {
    var params = new URLSearchParams(window.location.search);
    var dir = "/";
    if (params.has("dir")) {
        dir = params.get("dir");
    }
    if (dir.slice(-1) != "/") {
        dir += "/";
    }

    var datatable = $('#directory').DataTable({
        "columnDefs": [{
                "render": function (data, type, row) {
                    if (data[1]) {
                        // https://icons.getbootstrap.com/icons/folder/
                        return '<a href="/?dir=' + dir + data[0] + '">'
                        + '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-folder" viewBox="0 0 16 16"><path d="M.54 3.87.5 3a2 2 0 0 1 2-2h3.672a2 2 0 0 1 1.414.586l.828.828A2 2 0 0 0 9.828 3h3.982a2 2 0 0 1 1.992 2.181l-.637 7A2 2 0 0 1 13.174 14H2.826a2 2 0 0 1-1.991-1.819l-.637-7a1.99 1.99 0 0 1 .342-1.31zM2.19 4a1 1 0 0 0-.996 1.09l.637 7a1 1 0 0 0 .995.91h10.348a1 1 0 0 0 .995-.91l.637-7A1 1 0 0 0 13.81 4H2.19zm4.69-1.707A1 1 0 0 0 6.172 2H2.5a1 1 0 0 0-1 .981l.006.139C1.72 3.042 1.95 3 2.19 3h5.396l-.707-.707z"/></svg> '
                        + data[0] + '</a>';
                    }
                    return '<a href="#" onclick="fileInfo(\'' + data[0] + '\');">'
                        + '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-file-earmark" viewBox="0 0 16 16"><path d="M14 4.5V14a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V2a2 2 0 0 1 2-2h5.5L14 4.5zm-3 0A1.5 1.5 0 0 1 9.5 3V1H4a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1V4.5h-2z"/></svg> '
                        + data[0] + '</a>';
                },
                "targets": 0 // name
            },
            {
                "render": function (data, type, row) {
                    return new Date(Date.parse(data)).toLocaleString();
                },
                "targets": 1 // created_at
            },
            {
                "render": function (data, type, row) {
                    return humanFileSize(data);
                },
                "targets": 2 // size
            }
        ]
    });

    $.get("/api/list?dir=" + dir)
        .fail(function (xhr, status, error) {
            alert(xhr.responseText, 'danger');
        })
        .done(function (data) {
            data = data.split('\n');
            data.forEach(function (data) {
                try {
                    json = JSON.parse(data);
                    datatable.row.add([
                        [json.name, json.directory],
                        json.created_at,
                        json.size
                    ]).draw(false);
                } catch (e) {
                    console.log(e);
                }
            });

            datatable.draw();
        });

    // our whole tus party starts here..
    var upload = null
    var uploadIsRunning = false
    var toggleBtn = document.querySelector('#uploadButton')
    var input = document.querySelector('input[type=file]')
    var progress = document.querySelector('.progress')
    var progressBar = progress.querySelector('.progress-bar')
    var filePos = 0

    if (!tus.isSupported) {
        // if tus isn't supported we just bail out and it should default to the regular form upload option
        return
    }

    toggleBtn.addEventListener('click', (e) => {
        e.preventDefault()

        if (upload) {
            if (uploadIsRunning) {
                upload.abort()
                toggleBtn.textContent = 'resume upload'
                uploadIsRunning = false
            } else {
                upload.start()
                toggleBtn.textContent = 'pause upload'
                uploadIsRunning = true
            }
        } else if (input.files.length > 0) {
            startUpload()
        } else {
            input.click()
        }
    })

    input.addEventListener('change', startUpload)

    function startUpload() {
        var file = input.files[filePos]
        // Only continue if a file has actually been selected.
        // IE will trigger a change event even if we reset the input element
        // using reset() and we do not want to blow up later.
        if (!file) {
            filePos = 0
            input.value = ''
            return
        }

        progress.style.visibility = 'visible'
        progressBar.classList.remove("bg-warning")

        toggleBtn.textContent = 'pause upload'

        var options = {
            endpoint: "/api/upload?dir=" + dir,
            chunkSize: 1024 * 1024 * 32,
            retryDelays: [0, 1000, 3000, 5000],
            parallelUploads: 1,
            metadata: {
                filename: file.name,
                filetype: file.type,
            },
            onError(error) {
                progressBar.classList.add("bg-warning")
                if (error.originalRequest) {
                    if (window.confirm(`Failed because: ${error}\nDo you want to retry?`)) {
                        upload.start()
                        uploadIsRunning = true
                        return
                    }
                } else {
                    alert(`Failed because: ${error}`, "danger")
                }

                reset()
            },
            onProgress(bytesUploaded, bytesTotal) {
                var percentage = (bytesUploaded / bytesTotal * 100).toFixed(2)
                progressBar.style.width = `${percentage}%`
                console.log(bytesUploaded, bytesTotal, `${percentage}%`)
            },
            onSuccess() {
                progress.style.visibility = 'hidden'
                alert("Successfully uploaded " + file.name, "success")
                reset()
                filePos++
                startUpload()
            },
        }

        upload = new tus.Upload(file, options)
        upload.start()
        uploadIsRunning = true
    }

    function reset() {
        toggleBtn.textContent = 'start upload'
        upload = null
        uploadIsRunning = false
    }
});