// TODO: use decorators to simplify the internal fn behaviors?
function path_validator() {
  const [path, idx, line] = JSON.parse(Host.inputString())
  if(path.includes(".py")) {
    Host.outputString(JSON.stringify({value: true}))
  } else {
    Host.outputString(JSON.stringify({value: false}))
  }
}

function file_content_validator() {
  const [path, idx, file] = JSON.parse(Host.inputString())
  Host.outputString(JSON.stringify({value: false}))
}

function line_validator() {
  const [path, idx, line] = JSON.parse(Host.inputString())
  Host.outputString(JSON.stringify({value: false}))
}

module.exports = {path_validator, file_content_validator, line_validator}
