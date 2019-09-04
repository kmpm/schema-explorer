# SQL Schema Explorer

Copyright 2015-19 Tim Abell

[http://schemaexplorer.io/](http://schemaexplorer.io/)

SQL Schema Explorer is licenced under the [Affero-GPL v3](static/agpl-3.0.txt)

Included libraries remain under their [respective licenses](static/license.html)

# No Warranty

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

# Building

Make sure your `$GOPATH` is set (schema explorer hasn't been updated to use go modules yet).

Install [asdf](https://github.com/asdf-vm/asdf) to manage golang versions.

```bash
cd $GOPATH
mkdir -p src/github.com/timabell
cd src/github.com/timabell
git clone https://github.com/timabell/schema-explorer.git
cd schema-explorer
asdf plugin-add golang
asdf install golang 1.13
./build.sh
```
