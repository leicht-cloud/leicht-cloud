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
    var datatable = $('#directory').DataTable({
        "columnDefs": [{
                "render": function (data, type, row) {
                    console.log(data);
                    return '<a href="#" onclick="fileInfo(\'' + data + '\');">' + data + '</a>';
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

    $.get("/api/list")
        .fail(function (xhr, status, error) {
            alert(xhr.responseText, 'danger');
        })
        .done(function (data) {
            data = data.split('\n');
            data.forEach(function (data) {
                json = JSON.parse(data);
                datatable.row.add([
                    json.name,
                    json.created_at,
                    json.size
                ]).draw(false);
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

    if (!tus.isSupported) {
        alertBox.classList.remove('hidden')
    }

    if (!toggleBtn) {
        throw new Error('Toggle button not found on this page. Aborting upload-demo. ')
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
        var file = input.files[0]
        // Only continue if a file has actually been selected.
        // IE will trigger a change event even if we reset the input element
        // using reset() and we do not want to blow up later.
        if (!file) {
            return
        }

        progress.style.visibility = 'visible'
        progressBar.classList.remove("bg-success")
        progressBar.classList.remove("bg-warning")

        toggleBtn.textContent = 'pause upload'

        var options = {
            endpoint: "/api/upload",
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
                    window.alert(`Failed because: ${error}`)
                }

                reset()
            },
            onProgress(bytesUploaded, bytesTotal) {
                var percentage = (bytesUploaded / bytesTotal * 100).toFixed(2)
                progressBar.style.width = `${percentage}%`
                console.log(bytesUploaded, bytesTotal, `${percentage}%`)
            },
            onSuccess() {
                progressBar.classList.add("bg-success")
                reset()
            },
        }

        upload = new tus.Upload(file, options)
        upload.findPreviousUploads().then((previousUploads) => {
            askToResumeUpload(previousUploads, upload)

            upload.start()
            uploadIsRunning = true
        })
    }

    function reset() {
        input.value = ''
        toggleBtn.textContent = 'start upload'
        upload = null
        uploadIsRunning = false
    }

    function askToResumeUpload(previousUploads, upload) {
        if (previousUploads.length === 0) return

        let text = 'You tried to upload this file previously at these times:\n\n'
        previousUploads.forEach((previousUpload, index) => {
            text += `[${index}] ${previousUpload.creationTime}\n`
        })
        text += '\nEnter the corresponding number to resume an upload or press Cancel to start a new upload'

        const answer = prompt(text)
        const index = parseInt(answer, 10)

        if (!isNaN(index) && previousUploads[index]) {
            upload.resumeFromPreviousUpload(previousUploads[index])
        }
    }
});