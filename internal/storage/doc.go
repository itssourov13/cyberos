// Package storage contains persistence ports and adapters.
//
//	Domain / Application
//	        ↓
//	storage/ports (interfaces)
//	        ↓
//	storage/postgres (PostgreSQL adapter)
//
// Domain must never import this package or any SQL driver.
package storage
