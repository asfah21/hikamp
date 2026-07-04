# Project: Hikvision Broadcast Management System

## Overview

Transform the existing **MBTI** project into a completely new application named:

> Hikvision Broadcast Management System

The MBTI project will only be used as the **technical foundation**.

Keep all existing infrastructure, architecture, layout, middleware, authentication, Docker setup, Gin, HTMX, Alpine.js, sessions, routing style, flash messages, pagination, configuration, logging, helper functions, and project structure.

Remove all MBTI-related business logic and replace it with Broadcast Management.

This is **NOT** a refactor.

This is a complete domain conversion while preserving the architecture.

---

# Tech Stack

same as before
---

# UI Style

same as before
---

# Main Modules

## Dashboard

Display:

- Total Devices
- Online Devices
- Offline Devices
- Total Audio Files
- Today's Broadcast Count
- Next Scheduled Broadcast

Quick Action:

- Broadcast Now
- Sync Prayer Schedule
- Upload Audio

---

## Device Management

CRUD Devices

Fields:

- Name
- IP Address
- Port
- Username
- Password
- Location
- Status
- Firmware
- Last Sync
- Enabled

Test Connection button.

Read Device Information using:

GET

/ISAPI/System/deviceInfo

using HTTP Digest Authentication.

---

## Audio Library

Manage all custom audio.

Functions:

- Upload
- Rename
- Delete
- Search
- Sync to Device

Metadata:

- Name
- Category
- Duration
- File Size
- Device ID

Categories:

- Prayer
- Attendance
- Announcement
- Emergency
- Custom

---

## Broadcast Schedule

Core module.

Manage scheduled broadcasts.

Schedule Types:

- Daily
- Weekly
- Specific Date

Fields:

- Schedule Name
- Audio
- Device
- Begin Time
- End Time
- Volume
- Enabled

Support:

- Multiple Devices

Sync using Hikvision ISAPI.

Endpoint example:

POST

/ISAPI/VideoIntercom/broadcast/AddPlanScheme

Use JSON payload.

HTTP Digest Authentication.

---

## Manual Broadcast

Play audio immediately.

Select:

- Device
- Audio
- Volume

Click:

Broadcast Now

---

## Prayer Schedule

Generate prayer schedule automatically.

Store:

- Latitude
- Longitude
- Timezone

Generate:

- Fajr
- Dhuhr
- Asr
- Maghrib
- Isha

Allow:

Sync to Hikvision.

Automatically regenerate monthly.

---

## Broadcast Log

Record every broadcast.

Fields:

- Time
- Device
- Audio
- Result
- Duration
- Status
- Error Message

---

## Settings

General Settings

Prayer Settings

Company Name

Timezone

Default Volume

Default Device

Auto Sync

Dark Mode

---

# Hikvision Integration

Create package:

internal/hikvision/

Contains:

client.go

device.go

broadcast.go

audio.go

digest.go

terminal.go

Every communication must use:

HTTP Digest Authentication

Never duplicate Digest logic.

Create reusable client.

---

# Client Methods

DeviceInfo()

SearchAudio()

UploadAudio()

DeleteAudio()

SearchSchedule()

CreateSchedule()

UpdateSchedule()

DeleteSchedule()

BroadcastNow()

StopBroadcast()

TestConnection()

---

# Scheduler

Background scheduler.

Tasks:

Auto Prayer Sync

Auto Device Sync

Broadcast Cleanup

Log Cleanup

---

# Database

Tables

users

devices

audio_files

broadcast_schedules

broadcast_logs

settings

prayer_locations

---

# Dashboard Widgets

Device Status

Today's Broadcast

Next Broadcast

Prayer Times

Recent Logs

Quick Actions

---

# Design Principles

Keep controllers thin.

Business logic belongs in Services.

Database logic belongs in Repository.

Hikvision communication belongs in internal/hikvision.

Never place Hikvision HTTP calls inside controllers.

---

# HTMX

Every CRUD operation should use HTMX.

Partial rendering.

No full page refresh.

Use modal forms.

Use toast notifications.

---

# Error Handling

Consistent error responses.

Graceful UI.

Detailed logs.

---

# Docker

Keep existing Dockerfile.

Keep existing docker-compose.yml.

Only modify if necessary.

---

# Existing MBTI Code

Keep:

Authentication

Authorization

Middleware

Session

Layout

Sidebar

Navbar

Helpers

Logger

Configuration

Database

Docker

Project structure

Remove:

Every MBTI business logic

Every MBTI page

Every MBTI model

Every MBTI service

Every MBTI repository

Every MBTI route

Replace all MBTI naming with Broadcast Management.

---

# Future Ready

Design the application so future modules can be added easily:

- Live Broadcast
- Text To Speech
- Emergency Broadcast
- Multi Site
- Multi Company
- Odoo Integration
- Attendance Bell
- REST API
- MQTT
- WebSocket Monitoring

Use clean architecture while remaining a monolithic application.