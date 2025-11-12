# * Produced: Tuesday, November 11th, 2025
# * Author: Alec M.
# * GitHub: https://amattu.com/links/github
# * Copyright: (C) 2025 Alec M.
# * License: License GNU Affero General Public License v3.0
# *
# * This program is free software: you can redistribute it and/or modify
# * it under the terms of the GNU Affero General Public License as published by
# * the Free Software Foundation, either version 3 of the License, or
# * (at your option) any later version.
# *
# * This program is distributed in the hope that it will be useful,
# * but WITHOUT ANY WARRANTY; without even the implied warranty of
# * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# * GNU Affero General Public License for more details.
# *
# * You should have received a copy of the GNU Affero General Public License
# * along with this program.  If not, see <http://www.gnu.org/licenses/>.

#
# Variables
#
build_args = -a -o

#
# Targets
#

all: tests

# Run tests
tests:
	go test -v -cover ./...
