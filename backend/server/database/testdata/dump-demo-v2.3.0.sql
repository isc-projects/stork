--
-- PostgreSQL database dump
--

-- Dumped from database version 16.8 (Debian 16.8-1.pgdg120+1)
-- Dumped by pg_dump version 17.2 (Homebrew)

-- Started on 2025-10-31 11:00:44 CET

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;
SET unsupported_parameter = 'unsupported';  -- Simulates an unsupported parameter in this PostgreSQL version.

--
-- TOC entry 5 (class 2615 OID 2200)
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA public;


--
-- TOC entry 3950 (class 0 OID 0)
-- Dependencies: 5
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- TOC entry 1008 (class 1247 OID 16672)
-- Name: accesspointtype; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.accesspointtype AS ENUM (
    'control',
    'statistics'
);


--
-- TOC entry 984 (class 1247 OID 16520)
-- Name: hadhcptype; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.hadhcptype AS ENUM (
    'dhcp4',
    'dhcp6'
);


--
-- TOC entry 1029 (class 1247 OID 16769)
-- Name: hostdatasource; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.hostdatasource AS ENUM (
    'config',
    'api'
);


--
-- TOC entry 1023 (class 1247 OID 16740)
-- Name: hostidtype; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.hostidtype AS ENUM (
    'hw-address',
    'duid',
    'circuit-id',
    'client-id',
    'flex-id'
);


--
-- TOC entry 338 (class 1255 OID 16940)
-- Name: create_default_app_name(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.create_default_app_name() RETURNS trigger
    LANGUAGE plpgsql
    AS $_$
            BEGIN
                -- Trims whitespace before and after the actual name.
                SELECT REGEXP_REPLACE(NEW.name, '\s+$', '') INTO NEW.name;
                SELECT REGEXP_REPLACE(NEW.name, '^\s+', '') INTO NEW.name;

                IF NEW.name IS NULL OR NEW.name = '' THEN
                    -- Creates a base name without a postfix.
                    NEW.name = CONCAT(NEW.type, '@', (SELECT address FROM machine WHERE id = NEW.machine_id));
                    -- Checks whether the postfix is needed. It is necessary when the name already exists
                    -- without the postfix.
                    IF ((SELECT COUNT(*) FROM app WHERE name = NEW.name) > 0) THEN
                        NEW.name = CONCAT(NEW.name, '%', NEW.id::TEXT);
                    END IF;
                END IF;
            RETURN NEW;
            END;
            $_$;


--
-- TOC entry 348 (class 1255 OID 16980)
-- Name: delete_daemon_config_reports(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.delete_daemon_config_reports() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
            BEGIN
                DELETE FROM config_report AS c USING daemon_to_config_report AS d
                    WHERE c.id = d.config_report_id AND d.daemon_id = OLD.id;
                RETURN OLD;
            END;
            $$;


--
-- TOC entry 332 (class 1255 OID 16922)
-- Name: log_target_lower_severity(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.log_target_lower_severity() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
             BEGIN
                 NEW.severity = LOWER(NEW.severity);
                 RETURN NEW;
             END;
             $$;


--
-- TOC entry 329 (class 1255 OID 16669)
-- Name: match_subnet_network_family(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.match_subnet_network_family() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
             DECLARE
               net_family int;
             BEGIN
               IF NEW.shared_network_id IS NOT NULL THEN
                   net_family := (SELECT inet_family FROM shared_network WHERE id = NEW.shared_network_id);
                   IF net_family != family(NEW.prefix) THEN
                       RAISE EXCEPTION 'Family of the subnet % is not matching the shared network IPv% family', NEW.prefix, net_family;
                   END IF;
               END IF;
               RETURN NEW;
             END;
             $$;


--
-- TOC entry 347 (class 1255 OID 16944)
-- Name: replace_app_name(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.replace_app_name() RETURNS trigger
    LANGUAGE plpgsql
    AS $_$
            BEGIN
                -- Updates the app name only if the machine name was changed.
                IF NEW.address != OLD.address THEN
                    -- For each app following the pattern [text]@[machine-address], updates the
                    -- app name.
                    UPDATE app
                         SET name = REGEXP_REPLACE(name, CONCAT('@', OLD.address, '((\%\d+){0,1})$'), CONCAT('@', NEW.address, '\2'))
                    WHERE app.machine_id = NEW.id;
                END IF;
                RETURN NEW;
            END;
            $_$;


--
-- TOC entry 328 (class 1255 OID 16525)
-- Name: service_name_gen(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.service_name_gen() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
             BEGIN
                 -- This removes all whitespaces.
                 IF NEW.name IS NOT NULL THEN
	                 NEW.name = TRIM(NEW.name);
                 END IF;
                 IF NEW.name IS NULL OR NEW.name = '' THEN
                   NEW.name := 'service-' || to_char(NEW.id, 'FM0000000000');
                 END IF;
                 RETURN NEW;
             END;
             $$;


--
-- TOC entry 349 (class 1255 OID 17062)
-- Name: system_user_check_last_user(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.system_user_check_last_user() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
            DECLARE
              user_count integer;
              user_group integer;
            BEGIN
              SELECT group_id FROM system_user_to_group WHERE user_id = OLD.id INTO user_group;
              IF user_group != 1 THEN
                RETURN OLD;
              END IF;
              SELECT COUNT(*) FROM system_user_to_group WHERE group_id = 1 AND user_id != OLD.id LIMIT 1 INTO user_count;
              IF user_count = 0 THEN
                RAISE EXCEPTION 'deleting last super-admin user is forbidden';
              END IF;
              RETURN OLD;
            END;
            $$;


--
-- TOC entry 333 (class 1255 OID 16449)
-- Name: system_user_hash_password(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.system_user_hash_password() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
             BEGIN
               IF NEW.password_hash IS NOT NULL THEN
                 NEW.password_hash := crypt(NEW.password_hash, gen_salt('bf'));
               ELSIF OLD.password_hash IS NOT NULL THEN
                 NEW.password_hash := OLD.password_hash;
               END IF;
               RETURN NEW;
             END;
             $$;


--
-- TOC entry 330 (class 1255 OID 16699)
-- Name: update_machine_id(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_machine_id() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
              BEGIN
                  UPDATE app SET machine_id = NEW.machine_id WHERE id = NEW.app_id;
                  RETURN NEW;
              END;
              $$;


--
-- TOC entry 346 (class 1255 OID 16942)
-- Name: validate_app_name(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.validate_app_name() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
            DECLARE
                machine_name TEXT;
            BEGIN
                machine_name = SUBSTRING(NEW.name, CONCAT('@', '([^\%]+)'));
                IF machine_name IS NOT NULL AND STRPOS(machine_name, '@') = 0 THEN
                    IF ((SELECT COUNT(*) FROM machine WHERE address = machine_name) = 0) THEN
                         RAISE EXCEPTION 'machine % does not exist', machine_name;
                    END IF;
                END IF;
                RETURN NEW;
            END;
            $$;


--
-- TOC entry 331 (class 1255 OID 16893)
-- Name: wipe_dangling_service(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.wipe_dangling_service() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
            BEGIN
                DELETE FROM service
                    WHERE service.id = OLD.service_id AND NOT EXISTS (
                        SELECT FROM daemon_to_service AS ds
                            WHERE ds.service_id = service.id
                );
                RETURN NULL;
            END;
            $$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 244 (class 1259 OID 16677)
-- Name: access_point; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.access_point (
    app_id bigint NOT NULL,
    machine_id bigint NOT NULL,
    type public.accesspointtype NOT NULL,
    address text DEFAULT 'localhost'::text,
    port integer DEFAULT 0,
    key text DEFAULT ''::text,
    use_secure_protocol boolean DEFAULT false
);


--
-- TOC entry 240 (class 1259 OID 16611)
-- Name: address_pool; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.address_pool (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    lower_bound inet NOT NULL,
    upper_bound inet NOT NULL,
    kea_parameters jsonb,
    dhcp_option_set jsonb,
    dhcp_option_set_hash text,
    local_subnet_id bigint NOT NULL,
    stats jsonb,
    stats_collected_at timestamp without time zone,
    utilization smallint DEFAULT 0,
    CONSTRAINT address_pool_lower_upper_check CHECK ((lower_bound <= upper_bound)),
    CONSTRAINT address_pool_lower_upper_family_check CHECK ((family(lower_bound) = family(upper_bound))),
    CONSTRAINT stats_and_stats_collected_at_both_not_null CHECK (((stats IS NULL) = (stats_collected_at IS NULL)))
);


--
-- TOC entry 239 (class 1259 OID 16610)
-- Name: address_pool_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.address_pool_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3951 (class 0 OID 0)
-- Dependencies: 239
-- Name: address_pool_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.address_pool_id_seq OWNED BY public.address_pool.id;


--
-- TOC entry 224 (class 1259 OID 16465)
-- Name: app; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app (
    id integer NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'utc'::text) NOT NULL,
    machine_id integer NOT NULL,
    type character varying(10) NOT NULL,
    active boolean DEFAULT false,
    meta jsonb,
    details jsonb,
    name text
);


--
-- TOC entry 223 (class 1259 OID 16464)
-- Name: app_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.app_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3952 (class 0 OID 0)
-- Dependencies: 223
-- Name: app_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.app_id_seq OWNED BY public.app.id;


--
-- TOC entry 261 (class 1259 OID 16863)
-- Name: bind9_daemon; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bind9_daemon (
    id bigint NOT NULL,
    daemon_id bigint NOT NULL,
    stats jsonb
);


--
-- TOC entry 260 (class 1259 OID 16862)
-- Name: bind9_daemon_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.bind9_daemon_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3953 (class 0 OID 0)
-- Dependencies: 260
-- Name: bind9_daemon_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.bind9_daemon_id_seq OWNED BY public.bind9_daemon.id;


--
-- TOC entry 267 (class 1259 OID 16930)
-- Name: certs_serial_number_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.certs_serial_number_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 277 (class 1259 OID 17047)
-- Name: config_checker_preference; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_checker_preference (
    id bigint NOT NULL,
    daemon_id bigint,
    checker_name text NOT NULL,
    enabled boolean NOT NULL
);


--
-- TOC entry 276 (class 1259 OID 17046)
-- Name: config_checker_preference_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.config_checker_preference_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3954 (class 0 OID 0)
-- Dependencies: 276
-- Name: config_checker_preference_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.config_checker_preference_id_seq OWNED BY public.config_checker_preference.id;


--
-- TOC entry 270 (class 1259 OID 16950)
-- Name: config_report; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_report (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    checker_name text NOT NULL,
    content text,
    daemon_id bigint NOT NULL
);


--
-- TOC entry 269 (class 1259 OID 16949)
-- Name: config_report_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.config_report_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3955 (class 0 OID 0)
-- Dependencies: 269
-- Name: config_report_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.config_report_id_seq OWNED BY public.config_report.id;


--
-- TOC entry 273 (class 1259 OID 16984)
-- Name: config_review; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_review (
    id bigint NOT NULL,
    daemon_id bigint,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    config_hash text NOT NULL,
    signature text NOT NULL
);


--
-- TOC entry 272 (class 1259 OID 16983)
-- Name: config_review_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.config_review_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3956 (class 0 OID 0)
-- Dependencies: 272
-- Name: config_review_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.config_review_id_seq OWNED BY public.config_review.id;


--
-- TOC entry 255 (class 1259 OID 16815)
-- Name: daemon; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.daemon (
    id bigint NOT NULL,
    app_id bigint NOT NULL,
    pid integer,
    name text,
    active boolean DEFAULT false NOT NULL,
    version text,
    extended_version text,
    uptime bigint,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    reloaded_at timestamp without time zone,
    monitored boolean DEFAULT true
);


--
-- TOC entry 254 (class 1259 OID 16814)
-- Name: daemon_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.daemon_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3957 (class 0 OID 0)
-- Dependencies: 254
-- Name: daemon_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.daemon_id_seq OWNED BY public.daemon.id;


--
-- TOC entry 271 (class 1259 OID 16964)
-- Name: daemon_to_config_report; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.daemon_to_config_report (
    daemon_id bigint NOT NULL,
    config_report_id bigint NOT NULL,
    order_index bigint DEFAULT 0 NOT NULL
);


--
-- TOC entry 234 (class 1259 OID 16568)
-- Name: daemon_to_service; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.daemon_to_service (
    daemon_id bigint NOT NULL,
    service_id bigint NOT NULL
);


--
-- TOC entry 263 (class 1259 OID 16898)
-- Name: event; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event (
    id integer NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'utc'::text) NOT NULL,
    text text NOT NULL,
    level integer NOT NULL,
    relations jsonb,
    details text,
    sse_streams text[]
);


--
-- TOC entry 262 (class 1259 OID 16897)
-- Name: event_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.event_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3958 (class 0 OID 0)
-- Dependencies: 262
-- Name: event_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.event_id_seq OWNED BY public.event.id;


--
-- TOC entry 217 (class 1259 OID 16386)
-- Name: gopg_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.gopg_migrations (
    id integer NOT NULL,
    version bigint,
    created_at timestamp with time zone
);


--
-- TOC entry 216 (class 1259 OID 16385)
-- Name: gopg_migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.gopg_migrations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3959 (class 0 OID 0)
-- Dependencies: 216
-- Name: gopg_migrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.gopg_migrations_id_seq OWNED BY public.gopg_migrations.id;


--
-- TOC entry 233 (class 1259 OID 16541)
-- Name: ha_service; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ha_service (
    id bigint NOT NULL,
    service_id bigint NOT NULL,
    ha_type public.hadhcptype NOT NULL,
    ha_mode text,
    primary_id bigint,
    secondary_id bigint,
    primary_last_state text,
    secondary_last_state text,
    backup_id bigint[],
    primary_status_collected_at timestamp without time zone,
    secondary_status_collected_at timestamp without time zone,
    primary_last_scopes text[],
    secondary_last_scopes text[],
    primary_reachable boolean DEFAULT false,
    secondary_reachable boolean DEFAULT false,
    primary_last_failover_at timestamp without time zone,
    secondary_last_failover_at timestamp without time zone,
    primary_comm_interrupted boolean,
    primary_connecting_clients bigint,
    primary_unacked_clients bigint,
    primary_unacked_clients_left bigint,
    primary_analyzed_packets bigint,
    secondary_comm_interrupted boolean,
    secondary_connecting_clients bigint,
    secondary_unacked_clients bigint,
    secondary_unacked_clients_left bigint,
    secondary_analyzed_packets bigint,
    relationship text NOT NULL
);


--
-- TOC entry 232 (class 1259 OID 16540)
-- Name: ha_service_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ha_service_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3960 (class 0 OID 0)
-- Dependencies: 232
-- Name: ha_service_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ha_service_id_seq OWNED BY public.ha_service.id;


--
-- TOC entry 247 (class 1259 OID 16709)
-- Name: host; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.host (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    subnet_id bigint
);


--
-- TOC entry 246 (class 1259 OID 16708)
-- Name: host_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.host_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3961 (class 0 OID 0)
-- Dependencies: 246
-- Name: host_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.host_id_seq OWNED BY public.host.id;


--
-- TOC entry 251 (class 1259 OID 16752)
-- Name: host_identifier; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.host_identifier (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    type public.hostidtype NOT NULL,
    value bytea NOT NULL,
    host_id bigint NOT NULL
);


--
-- TOC entry 250 (class 1259 OID 16751)
-- Name: host_identifier_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.host_identifier_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3962 (class 0 OID 0)
-- Dependencies: 250
-- Name: host_identifier_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.host_identifier_id_seq OWNED BY public.host_identifier.id;


--
-- TOC entry 249 (class 1259 OID 16723)
-- Name: ip_reservation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ip_reservation (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    address cidr NOT NULL,
    local_host_id bigint NOT NULL
);


--
-- TOC entry 248 (class 1259 OID 16722)
-- Name: ip_reservation_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ip_reservation_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3963 (class 0 OID 0)
-- Dependencies: 248
-- Name: ip_reservation_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ip_reservation_id_seq OWNED BY public.ip_reservation.id;


--
-- TOC entry 257 (class 1259 OID 16831)
-- Name: kea_daemon; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.kea_daemon (
    id bigint NOT NULL,
    daemon_id bigint NOT NULL,
    config jsonb,
    config_hash text
);


--
-- TOC entry 256 (class 1259 OID 16830)
-- Name: kea_daemon_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.kea_daemon_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3964 (class 0 OID 0)
-- Dependencies: 256
-- Name: kea_daemon_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.kea_daemon_id_seq OWNED BY public.kea_daemon.id;


--
-- TOC entry 259 (class 1259 OID 16847)
-- Name: kea_dhcp_daemon; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.kea_dhcp_daemon (
    id bigint NOT NULL,
    kea_daemon_id bigint NOT NULL,
    stats jsonb
);


--
-- TOC entry 258 (class 1259 OID 16846)
-- Name: kea_dhcp_daemon_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.kea_dhcp_daemon_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3965 (class 0 OID 0)
-- Dependencies: 258
-- Name: kea_dhcp_daemon_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.kea_dhcp_daemon_id_seq OWNED BY public.kea_dhcp_daemon.id;


--
-- TOC entry 252 (class 1259 OID 16773)
-- Name: local_host; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.local_host (
    host_id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    data_source public.hostdatasource NOT NULL,
    daemon_id bigint NOT NULL,
    dhcp_option_set jsonb,
    dhcp_option_set_hash text,
    client_classes text[],
    next_server text,
    server_hostname text,
    boot_file_name text,
    id bigint NOT NULL,
    hostname text
);


--
-- TOC entry 281 (class 1259 OID 17133)
-- Name: local_host_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.local_host_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3966 (class 0 OID 0)
-- Dependencies: 281
-- Name: local_host_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.local_host_id_seq OWNED BY public.local_host.id;


--
-- TOC entry 278 (class 1259 OID 17069)
-- Name: local_shared_network; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.local_shared_network (
    daemon_id bigint NOT NULL,
    shared_network_id bigint NOT NULL,
    kea_parameters jsonb,
    dhcp_option_set jsonb,
    dhcp_option_set_hash text
);


--
-- TOC entry 243 (class 1259 OID 16644)
-- Name: local_subnet; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.local_subnet (
    subnet_id bigint NOT NULL,
    local_subnet_id bigint,
    stats_collected_at timestamp without time zone,
    stats jsonb,
    daemon_id bigint NOT NULL,
    kea_parameters jsonb,
    dhcp_option_set jsonb,
    dhcp_option_set_hash text,
    id bigint NOT NULL,
    user_context jsonb,
    CONSTRAINT stats_and_stats_collected_at_both_not_null CHECK ((((stats IS NOT NULL) AND (stats_collected_at IS NOT NULL)) OR ((stats IS NULL) AND (stats_collected_at IS NULL))))
);


--
-- TOC entry 279 (class 1259 OID 17088)
-- Name: local_subnet_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.local_subnet_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3967 (class 0 OID 0)
-- Dependencies: 279
-- Name: local_subnet_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.local_subnet_id_seq OWNED BY public.local_subnet.id;


--
-- TOC entry 285 (class 1259 OID 17166)
-- Name: local_zone; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.local_zone (
    id bigint NOT NULL,
    daemon_id bigint NOT NULL,
    zone_id bigint NOT NULL,
    view text DEFAULT '_default'::text NOT NULL,
    class text NOT NULL,
    serial bigint NOT NULL,
    type text NOT NULL,
    loaded_at timestamp without time zone NOT NULL,
    zone_transfer_at timestamp without time zone,
    rpz boolean DEFAULT false
);


--
-- TOC entry 284 (class 1259 OID 17165)
-- Name: local_zone_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.local_zone_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3968 (class 0 OID 0)
-- Dependencies: 284
-- Name: local_zone_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.local_zone_id_seq OWNED BY public.local_zone.id;


--
-- TOC entry 291 (class 1259 OID 17227)
-- Name: local_zone_rr; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.local_zone_rr (
    id bigint NOT NULL,
    local_zone_id bigint NOT NULL,
    name text NOT NULL,
    ttl bigint NOT NULL,
    class text NOT NULL,
    type text NOT NULL,
    rdata text NOT NULL
);


--
-- TOC entry 290 (class 1259 OID 17226)
-- Name: local_zone_rr_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.local_zone_rr_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3969 (class 0 OID 0)
-- Dependencies: 290
-- Name: local_zone_rr_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.local_zone_rr_id_seq OWNED BY public.local_zone_rr.id;


--
-- TOC entry 265 (class 1259 OID 16908)
-- Name: log_target; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.log_target (
    id bigint NOT NULL,
    daemon_id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    name text,
    severity text,
    output text NOT NULL
);


--
-- TOC entry 264 (class 1259 OID 16907)
-- Name: log_target_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.log_target_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3970 (class 0 OID 0)
-- Dependencies: 264
-- Name: log_target_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.log_target_id_seq OWNED BY public.log_target.id;


--
-- TOC entry 222 (class 1259 OID 16452)
-- Name: machine; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.machine (
    id integer NOT NULL,
    created_at timestamp without time zone DEFAULT (now() AT TIME ZONE 'utc'::text) NOT NULL,
    address character varying(255) NOT NULL,
    agent_port integer NOT NULL,
    state jsonb NOT NULL,
    last_visited_at timestamp without time zone,
    error character varying(255),
    agent_token text,
    cert_fingerprint bytea,
    authorized boolean DEFAULT false NOT NULL
);


--
-- TOC entry 221 (class 1259 OID 16451)
-- Name: machine_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.machine_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3971 (class 0 OID 0)
-- Dependencies: 221
-- Name: machine_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.machine_id_seq OWNED BY public.machine.id;


--
-- TOC entry 289 (class 1259 OID 17211)
-- Name: pdns_daemon; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pdns_daemon (
    id bigint NOT NULL,
    daemon_id bigint NOT NULL,
    details jsonb
);


--
-- TOC entry 288 (class 1259 OID 17210)
-- Name: pdns_daemon_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.pdns_daemon_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3972 (class 0 OID 0)
-- Dependencies: 288
-- Name: pdns_daemon_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.pdns_daemon_id_seq OWNED BY public.pdns_daemon.id;


--
-- TOC entry 242 (class 1259 OID 16628)
-- Name: prefix_pool; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.prefix_pool (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    prefix cidr NOT NULL,
    delegated_len smallint NOT NULL,
    excluded_prefix cidr,
    kea_parameters jsonb,
    dhcp_option_set jsonb,
    dhcp_option_set_hash text,
    local_subnet_id bigint NOT NULL,
    stats jsonb,
    stats_collected_at timestamp without time zone,
    utilization smallint DEFAULT 0,
    CONSTRAINT prefix_pool_delegated_len_check CHECK (((delegated_len > 0) AND (delegated_len <= 128))),
    CONSTRAINT prefix_pool_ipv6_only_check CHECK ((family((prefix)::inet) = 6)),
    CONSTRAINT stats_and_stats_collected_at_both_not_null CHECK (((stats IS NULL) = (stats_collected_at IS NULL)))
);


--
-- TOC entry 241 (class 1259 OID 16627)
-- Name: prefix_pool_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.prefix_pool_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3973 (class 0 OID 0)
-- Dependencies: 241
-- Name: prefix_pool_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.prefix_pool_id_seq OWNED BY public.prefix_pool.id;


--
-- TOC entry 266 (class 1259 OID 16924)
-- Name: rps_interval; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rps_interval (
    kea_daemon_id bigint NOT NULL,
    start_time timestamp without time zone NOT NULL,
    duration bigint,
    responses bigint
);


--
-- TOC entry 275 (class 1259 OID 17028)
-- Name: scheduled_config_change; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.scheduled_config_change (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    deadline_at timestamp without time zone NOT NULL,
    executed boolean DEFAULT false,
    error text,
    user_id bigint NOT NULL,
    updates jsonb NOT NULL
);


--
-- TOC entry 274 (class 1259 OID 17027)
-- Name: scheduled_config_change_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.scheduled_config_change_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3974 (class 0 OID 0)
-- Dependencies: 274
-- Name: scheduled_config_change_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.scheduled_config_change_id_seq OWNED BY public.scheduled_config_change.id;


--
-- TOC entry 268 (class 1259 OID 16931)
-- Name: secret; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.secret (
    name text NOT NULL,
    content text NOT NULL
);


--
-- TOC entry 231 (class 1259 OID 16527)
-- Name: service; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.service (
    id bigint NOT NULL,
    name text,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    CONSTRAINT service_name_not_blank CHECK (((name IS NOT NULL) AND (btrim(name) <> ''::text)))
);


--
-- TOC entry 230 (class 1259 OID 16526)
-- Name: service_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.service_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3975 (class 0 OID 0)
-- Dependencies: 230
-- Name: service_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.service_id_seq OWNED BY public.service.id;


--
-- TOC entry 218 (class 1259 OID 16390)
-- Name: sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sessions (
    token text NOT NULL,
    data bytea NOT NULL,
    expiry timestamp with time zone NOT NULL
);


--
-- TOC entry 3976 (class 0 OID 0)
-- Dependencies: 218
-- Name: TABLE sessions; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sessions IS 'Table storing sessions according for scs.';


--
-- TOC entry 245 (class 1259 OID 16701)
-- Name: setting; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.setting (
    name text NOT NULL,
    val_type integer NOT NULL,
    value text NOT NULL
);


--
-- TOC entry 236 (class 1259 OID 16585)
-- Name: shared_network; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shared_network (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    name text NOT NULL,
    inet_family integer NOT NULL,
    addr_utilization smallint,
    pd_utilization smallint,
    stats jsonb,
    stats_collected_at timestamp without time zone,
    out_of_pool_addr_utilization smallint,
    out_of_pool_pd_utilization smallint,
    CONSTRAINT shared_network_family_46 CHECK (((inet_family = 4) OR (inet_family = 6)))
);


--
-- TOC entry 235 (class 1259 OID 16584)
-- Name: shared_network_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.shared_network_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3977 (class 0 OID 0)
-- Dependencies: 235
-- Name: shared_network_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.shared_network_id_seq OWNED BY public.shared_network.id;


--
-- TOC entry 253 (class 1259 OID 16795)
-- Name: statistic; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.statistic (
    name text NOT NULL,
    value numeric(60,0)
);


--
-- TOC entry 238 (class 1259 OID 16595)
-- Name: subnet; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subnet (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    prefix cidr NOT NULL,
    shared_network_id bigint,
    client_class text,
    addr_utilization smallint,
    pd_utilization smallint,
    stats jsonb,
    stats_collected_at timestamp without time zone,
    out_of_pool_addr_utilization smallint,
    out_of_pool_pd_utilization smallint
);


--
-- TOC entry 237 (class 1259 OID 16594)
-- Name: subnet_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.subnet_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3978 (class 0 OID 0)
-- Dependencies: 237
-- Name: subnet_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.subnet_id_seq OWNED BY public.subnet.id;


--
-- TOC entry 226 (class 1259 OID 16485)
-- Name: system_group; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_group (
    id bigint NOT NULL,
    name text,
    description text
);


--
-- TOC entry 225 (class 1259 OID 16484)
-- Name: system_group_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.system_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3979 (class 0 OID 0)
-- Dependencies: 225
-- Name: system_group_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.system_group_id_seq OWNED BY public.system_group.id;


--
-- TOC entry 219 (class 1259 OID 16395)
-- Name: system_user; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public."system_user" (
    id integer NOT NULL,
    email text,
    lastname text,
    name text,
    login text,
    auth_method text DEFAULT 'internal'::text NOT NULL,
    external_id text,
    change_password boolean DEFAULT false NOT NULL,
    CONSTRAINT system_user_external_id_required_for_external_users CHECK (((auth_method = 'internal'::text) = (external_id IS NULL))),
    CONSTRAINT system_user_login_email_exist_check CHECK (((login IS NOT NULL) OR (email IS NOT NULL)))
);


--
-- TOC entry 3980 (class 0 OID 0)
-- Dependencies: 219
-- Name: TABLE "system_user"; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public."system_user" IS 'Table holding a list of users which are known to the system.';


--
-- TOC entry 220 (class 1259 OID 16400)
-- Name: system_user_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.system_user_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3981 (class 0 OID 0)
-- Dependencies: 220
-- Name: system_user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.system_user_id_seq OWNED BY public."system_user".id;


--
-- TOC entry 280 (class 1259 OID 17110)
-- Name: system_user_password; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_user_password (
    id integer NOT NULL,
    password_hash text NOT NULL
);


--
-- TOC entry 229 (class 1259 OID 16495)
-- Name: system_user_to_group; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system_user_to_group (
    user_id bigint NOT NULL,
    group_id bigint NOT NULL
);


--
-- TOC entry 228 (class 1259 OID 16494)
-- Name: system_user_to_group_group_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.system_user_to_group_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3982 (class 0 OID 0)
-- Dependencies: 228
-- Name: system_user_to_group_group_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.system_user_to_group_group_id_seq OWNED BY public.system_user_to_group.group_id;


--
-- TOC entry 227 (class 1259 OID 16493)
-- Name: system_user_to_group_user_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.system_user_to_group_user_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3983 (class 0 OID 0)
-- Dependencies: 227
-- Name: system_user_to_group_user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.system_user_to_group_user_id_seq OWNED BY public.system_user_to_group.user_id;


--
-- TOC entry 283 (class 1259 OID 17155)
-- Name: zone; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.zone (
    id bigint NOT NULL,
    name text NOT NULL,
    rname text NOT NULL
);


--
-- TOC entry 282 (class 1259 OID 17154)
-- Name: zone_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.zone_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3984 (class 0 OID 0)
-- Dependencies: 282
-- Name: zone_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.zone_id_seq OWNED BY public.zone.id;


--
-- TOC entry 287 (class 1259 OID 17190)
-- Name: zone_inventory_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.zone_inventory_state (
    id bigint NOT NULL,
    daemon_id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL,
    state jsonb NOT NULL
);


--
-- TOC entry 286 (class 1259 OID 17189)
-- Name: zone_inventory_state_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.zone_inventory_state_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3985 (class 0 OID 0)
-- Dependencies: 286
-- Name: zone_inventory_state_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.zone_inventory_state_id_seq OWNED BY public.zone_inventory_state.id;


--
-- TOC entry 3482 (class 2604 OID 16614)
-- Name: address_pool id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.address_pool ALTER COLUMN id SET DEFAULT nextval('public.address_pool_id_seq'::regclass);


--
-- TOC entry 3467 (class 2604 OID 16468)
-- Name: app id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app ALTER COLUMN id SET DEFAULT nextval('public.app_id_seq'::regclass);


--
-- TOC entry 3507 (class 2604 OID 16866)
-- Name: bind9_daemon id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bind9_daemon ALTER COLUMN id SET DEFAULT nextval('public.bind9_daemon_id_seq'::regclass);


--
-- TOC entry 3520 (class 2604 OID 17050)
-- Name: config_checker_preference id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_checker_preference ALTER COLUMN id SET DEFAULT nextval('public.config_checker_preference_id_seq'::regclass);


--
-- TOC entry 3512 (class 2604 OID 16953)
-- Name: config_report id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_report ALTER COLUMN id SET DEFAULT nextval('public.config_report_id_seq'::regclass);


--
-- TOC entry 3515 (class 2604 OID 16987)
-- Name: config_review id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_review ALTER COLUMN id SET DEFAULT nextval('public.config_review_id_seq'::regclass);


--
-- TOC entry 3501 (class 2604 OID 16818)
-- Name: daemon id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon ALTER COLUMN id SET DEFAULT nextval('public.daemon_id_seq'::regclass);


--
-- TOC entry 3508 (class 2604 OID 16901)
-- Name: event id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event ALTER COLUMN id SET DEFAULT nextval('public.event_id_seq'::regclass);


--
-- TOC entry 3460 (class 2604 OID 16389)
-- Name: gopg_migrations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gopg_migrations ALTER COLUMN id SET DEFAULT nextval('public.gopg_migrations_id_seq'::regclass);


--
-- TOC entry 3475 (class 2604 OID 16544)
-- Name: ha_service id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ha_service ALTER COLUMN id SET DEFAULT nextval('public.ha_service_id_seq'::regclass);


--
-- TOC entry 3493 (class 2604 OID 16712)
-- Name: host id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.host ALTER COLUMN id SET DEFAULT nextval('public.host_id_seq'::regclass);


--
-- TOC entry 3497 (class 2604 OID 16755)
-- Name: host_identifier id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.host_identifier ALTER COLUMN id SET DEFAULT nextval('public.host_identifier_id_seq'::regclass);


--
-- TOC entry 3495 (class 2604 OID 16726)
-- Name: ip_reservation id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ip_reservation ALTER COLUMN id SET DEFAULT nextval('public.ip_reservation_id_seq'::regclass);


--
-- TOC entry 3505 (class 2604 OID 16834)
-- Name: kea_daemon id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kea_daemon ALTER COLUMN id SET DEFAULT nextval('public.kea_daemon_id_seq'::regclass);


--
-- TOC entry 3506 (class 2604 OID 16850)
-- Name: kea_dhcp_daemon id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kea_dhcp_daemon ALTER COLUMN id SET DEFAULT nextval('public.kea_dhcp_daemon_id_seq'::regclass);


--
-- TOC entry 3500 (class 2604 OID 17134)
-- Name: local_host id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_host ALTER COLUMN id SET DEFAULT nextval('public.local_host_id_seq'::regclass);


--
-- TOC entry 3488 (class 2604 OID 17089)
-- Name: local_subnet id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_subnet ALTER COLUMN id SET DEFAULT nextval('public.local_subnet_id_seq'::regclass);


--
-- TOC entry 3522 (class 2604 OID 17169)
-- Name: local_zone id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_zone ALTER COLUMN id SET DEFAULT nextval('public.local_zone_id_seq'::regclass);


--
-- TOC entry 3528 (class 2604 OID 17230)
-- Name: local_zone_rr id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_zone_rr ALTER COLUMN id SET DEFAULT nextval('public.local_zone_rr_id_seq'::regclass);


--
-- TOC entry 3510 (class 2604 OID 16911)
-- Name: log_target id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.log_target ALTER COLUMN id SET DEFAULT nextval('public.log_target_id_seq'::regclass);


--
-- TOC entry 3464 (class 2604 OID 16455)
-- Name: machine id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.machine ALTER COLUMN id SET DEFAULT nextval('public.machine_id_seq'::regclass);


--
-- TOC entry 3527 (class 2604 OID 17214)
-- Name: pdns_daemon id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pdns_daemon ALTER COLUMN id SET DEFAULT nextval('public.pdns_daemon_id_seq'::regclass);


--
-- TOC entry 3485 (class 2604 OID 16631)
-- Name: prefix_pool id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prefix_pool ALTER COLUMN id SET DEFAULT nextval('public.prefix_pool_id_seq'::regclass);


--
-- TOC entry 3517 (class 2604 OID 17031)
-- Name: scheduled_config_change id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.scheduled_config_change ALTER COLUMN id SET DEFAULT nextval('public.scheduled_config_change_id_seq'::regclass);


--
-- TOC entry 3473 (class 2604 OID 16530)
-- Name: service id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.service ALTER COLUMN id SET DEFAULT nextval('public.service_id_seq'::regclass);


--
-- TOC entry 3478 (class 2604 OID 16588)
-- Name: shared_network id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shared_network ALTER COLUMN id SET DEFAULT nextval('public.shared_network_id_seq'::regclass);


--
-- TOC entry 3480 (class 2604 OID 16598)
-- Name: subnet id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subnet ALTER COLUMN id SET DEFAULT nextval('public.subnet_id_seq'::regclass);


--
-- TOC entry 3470 (class 2604 OID 16488)
-- Name: system_group id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_group ALTER COLUMN id SET DEFAULT nextval('public.system_group_id_seq'::regclass);


--
-- TOC entry 3461 (class 2604 OID 16401)
-- Name: system_user id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."system_user" ALTER COLUMN id SET DEFAULT nextval('public.system_user_id_seq'::regclass);


--
-- TOC entry 3471 (class 2604 OID 16498)
-- Name: system_user_to_group user_id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user_to_group ALTER COLUMN user_id SET DEFAULT nextval('public.system_user_to_group_user_id_seq'::regclass);


--
-- TOC entry 3472 (class 2604 OID 16499)
-- Name: system_user_to_group group_id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user_to_group ALTER COLUMN group_id SET DEFAULT nextval('public.system_user_to_group_group_id_seq'::regclass);


--
-- TOC entry 3521 (class 2604 OID 17158)
-- Name: zone id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.zone ALTER COLUMN id SET DEFAULT nextval('public.zone_id_seq'::regclass);


--
-- TOC entry 3525 (class 2604 OID 17193)
-- Name: zone_inventory_state id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.zone_inventory_state ALTER COLUMN id SET DEFAULT nextval('public.zone_inventory_state_id_seq'::regclass);


--
-- TOC entry 3897 (class 0 OID 16677)
-- Dependencies: 244
-- Data for Name: access_point; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.access_point VALUES (3, 2, 'control', '127.0.0.1', 8000, '', false);
INSERT INTO public.access_point VALUES (2, 1, 'control', '127.0.0.1', 8000, '', false);
INSERT INTO public.access_point VALUES (6, 7, 'control', '127.0.0.1', 8085, 'stork', false);
INSERT INTO public.access_point VALUES (5, 5, 'control', '127.0.0.1', 8001, '', false);
INSERT INTO public.access_point VALUES (7, 9, 'control', '127.0.0.1', 953, 'rndc-key:hmac-sha256:C0WsVMnbpYt3RxJEZCrmJmlRyQJp9vy2lKp887r19mY=', false);
INSERT INTO public.access_point VALUES (7, 9, 'statistics', '127.0.0.1', 8053, '', false);
INSERT INTO public.access_point VALUES (4, 3, 'control', '127.0.0.1', 8002, '', false);
INSERT INTO public.access_point VALUES (1, 4, 'control', '127.0.0.1', 8000, '', false);
INSERT INTO public.access_point VALUES (8, 8, 'control', '127.0.0.1', 953, 'rndc-key:hmac-sha256:C0WsVMnbpYt3RxJEZCrmJmlRyQJp9vy2lKp887r19mY=', false);
INSERT INTO public.access_point VALUES (8, 8, 'statistics', '127.0.0.1', 8053, '', false);


--
-- TOC entry 3893 (class 0 OID 16611)
-- Dependencies: 240
-- Data for Name: address_pool; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.address_pool VALUES (13, '2025-10-31 09:53:44.651209', '192.1.15.1', '192.1.15.50', '{"pool-id": 1015001}', NULL, NULL, 8, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.02068', 0);
INSERT INTO public.address_pool VALUES (56, '2025-10-31 09:53:46.547234', '192.0.3.1', '192.0.3.200', '{}', NULL, NULL, 28, '{"total-addresses": "200", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.050263', 0);
INSERT INTO public.address_pool VALUES (2, '2025-10-31 09:53:44.651209', '192.0.6.1', '192.0.6.40', '{"pool-id": 6001}', NULL, NULL, 2, '{"total-addresses": "40", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.020893', 0);
INSERT INTO public.address_pool VALUES (12, '2025-10-31 09:53:44.651209', '192.0.10.84', '192.0.10.84', '{"pool-id": 10082}', NULL, NULL, 7, '{"total-addresses": "3", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.021876', 0);
INSERT INTO public.address_pool VALUES (55, '2025-10-31 09:53:46.547234', '192.0.20.1', '192.0.20.200', '{}', NULL, NULL, 27, '{"total-addresses": "200", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.05048', 0);
INSERT INTO public.address_pool VALUES (10, '2025-10-31 09:53:44.651209', '192.0.10.82', '192.0.10.82', '{"pool-id": 10082}', NULL, NULL, 7, '{"total-addresses": "3", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.022064', 0);
INSERT INTO public.address_pool VALUES (57, '2025-10-31 09:53:47.037418', '192.0.3.1', '192.0.3.200', '{}', NULL, NULL, 31, '{"total-addresses": "200", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.056582', 0);
INSERT INTO public.address_pool VALUES (21, '2025-10-31 09:53:44.651209', '192.1.17.81', '192.1.17.100', '{"pool-id": 1017081}', NULL, NULL, 10, '{"total-addresses": "20", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.016235', 0);
INSERT INTO public.address_pool VALUES (20, '2025-10-31 09:53:44.651209', '192.1.17.66', '192.1.17.80', '{"pool-id": 1017066}', NULL, NULL, 10, '{"total-addresses": "15", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.016425', 0);
INSERT INTO public.address_pool VALUES (27, '2025-10-31 09:53:44.651209', '192.1.17.201', '192.1.17.220', '{"pool-id": 1017181}', NULL, NULL, 10, '{"total-addresses": "60", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.016861', 0);
INSERT INTO public.address_pool VALUES (11, '2025-10-31 09:53:44.651209', '192.0.10.83', '192.0.10.83', '{"pool-id": 10082}', NULL, NULL, 7, '{"total-addresses": "3", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.022349', 0);
INSERT INTO public.address_pool VALUES (9, '2025-10-31 09:53:44.651209', '192.0.10.1', '192.0.10.50', '{"pool-id": 10001}', NULL, NULL, 6, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.01935', 0);
INSERT INTO public.address_pool VALUES (14, '2025-10-31 09:53:44.651209', '192.1.16.1', '192.1.16.50', '{"pool-id": 1016001}', NULL, NULL, 9, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.019779', 0);
INSERT INTO public.address_pool VALUES (54, '2025-10-31 09:53:45.842424', '192.0.20.1', '192.0.20.200', '{}', NULL, NULL, 24, '{"total-addresses": "200", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.044137', 0);
INSERT INTO public.address_pool VALUES (16, '2025-10-31 09:53:44.651209', '192.1.16.101', '192.1.16.150', '{"pool-id": 1016101}', NULL, NULL, 9, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.020215', 0);
INSERT INTO public.address_pool VALUES (30, '2025-10-31 09:53:44.651209', '192.1.17.244', '192.1.17.246', '{"pool-id": 1017241}', NULL, NULL, 10, '{"total-addresses": "10", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.015141', 0);
INSERT INTO public.address_pool VALUES (18, '2025-10-31 09:53:44.651209', '192.1.17.21', '192.1.17.40', '{"pool-id": 1017021}', NULL, NULL, 10, '{"total-addresses": "20", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.015855', 0);
INSERT INTO public.address_pool VALUES (35, '2025-10-31 09:53:44.651209', '192.0.2.151', '192.0.2.200', '{}', NULL, NULL, 11, '{"total-addresses": "200", "reclaimed-leases": "0", "assigned-addresses": "2", "declined-addresses": "1", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.018697', 10);
INSERT INTO public.address_pool VALUES (32, '2025-10-31 09:53:44.651209', '192.0.2.1', '192.0.2.50', '{}', NULL, NULL, 11, '{"total-addresses": "200", "reclaimed-leases": "0", "assigned-addresses": "2", "declined-addresses": "1", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.018918', 10);
INSERT INTO public.address_pool VALUES (29, '2025-10-31 09:53:44.651209', '192.1.17.241', '192.1.17.243', '{"pool-id": 1017241}', NULL, NULL, 10, '{"total-addresses": "10", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.015568', 0);
INSERT INTO public.address_pool VALUES (33, '2025-10-31 09:53:44.651209', '192.0.2.51', '192.0.2.100', '{}', NULL, NULL, 11, '{"total-addresses": "200", "reclaimed-leases": "0", "assigned-addresses": "2", "declined-addresses": "1", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.019153', 10);
INSERT INTO public.address_pool VALUES (17, '2025-10-31 09:53:44.651209', '192.1.17.1', '192.1.17.20', '{"pool-id": 1017001}', NULL, NULL, 10, '{"total-addresses": "20", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.016049', 0);
INSERT INTO public.address_pool VALUES (8, '2025-10-31 09:53:44.651209', '192.0.9.1', '192.0.9.50', '{"pool-id": 9001}', NULL, NULL, 5, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.019576', 0);
INSERT INTO public.address_pool VALUES (5, '2025-10-31 09:53:44.651209', '192.0.7.1', '192.0.7.50', '{"pool-id": 7001}', NULL, NULL, 3, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.021494', 0);
INSERT INTO public.address_pool VALUES (15, '2025-10-31 09:53:44.651209', '192.1.16.51', '192.1.16.100', '{"pool-id": 1016051}', NULL, NULL, 9, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.019987', 0);
INSERT INTO public.address_pool VALUES (31, '2025-10-31 09:53:44.651209', '192.1.17.247', '192.1.17.250', '{"pool-id": 1017241}', NULL, NULL, 10, '{"total-addresses": "10", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.01535', 0);
INSERT INTO public.address_pool VALUES (6, '2025-10-31 09:53:44.651209', '192.0.7.51', '192.0.7.100', '{"pool-id": 7051}', NULL, NULL, 3, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.021689', 0);
INSERT INTO public.address_pool VALUES (4, '2025-10-31 09:53:44.651209', '192.0.6.111', '192.0.6.150', '{"pool-id": 6111}', NULL, NULL, 2, '{"total-addresses": "40", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.021095', 0);
INSERT INTO public.address_pool VALUES (1, '2025-10-31 09:53:44.651209', '192.0.5.1', '192.0.5.50', '{}', NULL, NULL, 1, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.014899', 0);
INSERT INTO public.address_pool VALUES (42, '2025-10-31 09:53:45.426795', '5002:dba::', '5002:dba::ffff:ffff:ffff:ffff', '{"pool-id": 602}', NULL, NULL, 15, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.036683', 0);
INSERT INTO public.address_pool VALUES (41, '2025-10-31 09:53:45.426795', '5002:db8::', '5002:db8::ffff:ffff:ffff:ffff', '{"pool-id": 601}', NULL, NULL, 15, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.036874', 0);
INSERT INTO public.address_pool VALUES (3, '2025-10-31 09:53:44.651209', '192.0.6.61', '192.0.6.90', '{"pool-id": 6061}', NULL, NULL, 2, '{"total-addresses": "30", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.021298', 0);
INSERT INTO public.address_pool VALUES (38, '2025-10-31 09:53:45.426795', '5000:dba::', '5000:dba::ffff:ffff:ffff:ffff', '{"pool-id": 402}', NULL, NULL, 13, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.037087', 0);
INSERT INTO public.address_pool VALUES (37, '2025-10-31 09:53:45.426795', '5000:db8::', '5000:db8::ffff:ffff:ffff:ffff', '{"pool-id": 401}', NULL, NULL, 13, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.037283', 0);
INSERT INTO public.address_pool VALUES (40, '2025-10-31 09:53:45.426795', '5001:dba::', '5001:dba::ffff:ffff:ffff:ffff', '{"pool-id": 502}', NULL, NULL, 14, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.035326', 0);
INSERT INTO public.address_pool VALUES (51, '2025-10-31 09:53:45.426795', '3001:db8:1:0:3::', '3001:db8:1:0:3:ffff:ffff:ffff', '{}', NULL, NULL, 19, '{"total-nas": "844424930131968", "assigned-nas": "2", "declined-nas": "1", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.035879', 0);
INSERT INTO public.address_pool VALUES (44, '2025-10-31 09:53:45.426795', '5003:dba::', '5003:dba::ffff:ffff:ffff:ffff', '{"pool-id": 702}', NULL, NULL, 16, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.037461', 0);
INSERT INTO public.address_pool VALUES (28, '2025-10-31 09:53:44.651209', '192.1.17.221', '192.1.17.240', '{"pool-id": 1017181}', NULL, NULL, 10, '{"total-addresses": "60", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.017068', 0);
INSERT INTO public.address_pool VALUES (26, '2025-10-31 09:53:44.651209', '192.1.17.181', '192.1.17.200', '{"pool-id": 1017181}', NULL, NULL, 10, '{"total-addresses": "60", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.017314', 0);
INSERT INTO public.address_pool VALUES (25, '2025-10-31 09:53:44.651209', '192.1.17.161', '192.1.17.180', '{"pool-id": 1017161}', NULL, NULL, 10, '{"total-addresses": "20", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.017537', 0);
INSERT INTO public.address_pool VALUES (49, '2025-10-31 09:53:45.426795', '3001:db8:1:0:1::', '3001:db8:1:0:1:ffff:ffff:ffff', '{}', NULL, NULL, 19, '{"total-nas": "844424930131968", "assigned-nas": "2", "declined-nas": "1", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.036054', 0);
INSERT INTO public.address_pool VALUES (47, '2025-10-31 09:53:45.426795', '5005:db8::', '5005:db8::ffff:ffff:ffff:ffff', '{"pool-id": 901}', NULL, NULL, 18, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.034857', 0);
INSERT INTO public.address_pool VALUES (19, '2025-10-31 09:53:44.651209', '192.1.17.41', '192.1.17.60', '{"pool-id": 1017041}', NULL, NULL, 10, '{"total-addresses": "20", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.017736', 0);
INSERT INTO public.address_pool VALUES (24, '2025-10-31 09:53:44.651209', '192.1.17.141', '192.1.17.160', '{"pool-id": 1017141}', NULL, NULL, 10, '{"total-addresses": "20", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.01794', 0);
INSERT INTO public.address_pool VALUES (43, '2025-10-31 09:53:45.426795', '5003:db8::', '5003:db8::ffff:ffff:ffff:ffff', '{"pool-id": 701}', NULL, NULL, 16, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.03764', 0);
INSERT INTO public.address_pool VALUES (45, '2025-10-31 09:53:45.426795', '5004:db8::', '5004:db8::ffff:ffff:ffff:ffff', '{"pool-id": 801}', NULL, NULL, 17, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.037818', 0);
INSERT INTO public.address_pool VALUES (22, '2025-10-31 09:53:44.651209', '192.1.17.101', '192.1.17.120', '{"pool-id": 1017101}', NULL, NULL, 10, '{"total-addresses": "20", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.018214', 0);
INSERT INTO public.address_pool VALUES (46, '2025-10-31 09:53:45.426795', '5004:dba::', '5004:dba::ffff:ffff:ffff:ffff', '{"pool-id": 802}', NULL, NULL, 17, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.037992', 0);
INSERT INTO public.address_pool VALUES (7, '2025-10-31 09:53:44.651209', '192.0.8.1', '192.0.8.50', '{"pool-id": 8001}', NULL, NULL, 4, '{"total-addresses": "50", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.020464', 0);
INSERT INTO public.address_pool VALUES (48, '2025-10-31 09:53:45.426795', '5005:dba::', '5005:dba::ffff:ffff:ffff:ffff', '{"pool-id": 902}', NULL, NULL, 18, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.035113', 0);
INSERT INTO public.address_pool VALUES (50, '2025-10-31 09:53:45.426795', '3001:db8:1:0:2::', '3001:db8:1:0:2:ffff:ffff:ffff', '{}', NULL, NULL, 19, '{"total-nas": "844424930131968", "assigned-nas": "2", "declined-nas": "1", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.0357', 0);
INSERT INTO public.address_pool VALUES (39, '2025-10-31 09:53:45.426795', '5001:db8::', '5001:db8::ffff:ffff:ffff:ffff', '{"pool-id": 501}', NULL, NULL, 14, '{"total-nas": "18446744073709551616", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.03552', 0);
INSERT INTO public.address_pool VALUES (52, '2025-10-31 09:53:45.426795', '3000:db8:1::', '3000:db8:1::ffff:ffff:ffff', '{}', NULL, NULL, 20, '{"total-nas": "281474976710656", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.034633', 0);
INSERT INTO public.address_pool VALUES (36, '2025-10-31 09:53:45.426795', '4001:db8:1:0:abcd::', '4001:db8:1:0:abcd:ffff:ffff:ffff', '{"pool-id": 301}', NULL, NULL, 12, '{"total-nas": "281474976710656", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.036462', 0);
INSERT INTO public.address_pool VALUES (23, '2025-10-31 09:53:44.651209', '192.1.17.121', '192.1.17.140', '{"pool-id": 1017121}', NULL, NULL, 10, '{"total-addresses": "20", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.016612', 0);
INSERT INTO public.address_pool VALUES (34, '2025-10-31 09:53:44.651209', '192.0.2.101', '192.0.2.150', '{}', NULL, NULL, 11, '{"total-addresses": "200", "reclaimed-leases": "0", "assigned-addresses": "2", "declined-addresses": "1", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', '2025-10-31 09:59:57.018433', 10);
INSERT INTO public.address_pool VALUES (53, '2025-10-31 09:53:45.426795', '3001:1234:5678:90ab:cdef:1f2e:3d4c:5b68', '3001:1234:5678:90ab:cdef:1f2e:3d4c:5b6b', '{}', NULL, NULL, 21, '{"total-nas": "4", "assigned-nas": "0", "declined-nas": "0", "reclaimed-leases": "0", "cumulative-assigned-nas": "0", "reclaimed-declined-addresses": "0"}', '2025-10-31 09:59:57.036233', 0);


--
-- TOC entry 3877 (class 0 OID 16465)
-- Dependencies: 224
-- Data for Name: app; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.app VALUES (3, '2025-10-31 09:53:45.842424', 2, 'kea', NULL, '{"Version": "3.1.0", "ExtendedVersion": ""}', NULL, 'kea@agent-kea-ha3');
INSERT INTO public.app VALUES (2, '2025-10-31 09:53:45.426795', 1, 'kea', true, '{"Version": "3.1.0", "ExtendedVersion": ""}', NULL, 'kea@agent-kea6');
INSERT INTO public.app VALUES (6, '2025-10-31 09:53:47.874309', 7, 'pdns', true, '{"Version": "4.7.3", "ExtendedVersion": "4.7.3"}', NULL, 'pdns@agent-pdns');
INSERT INTO public.app VALUES (5, '2025-10-31 09:53:47.037418', 5, 'kea', NULL, '{"Version": "3.1.0", "ExtendedVersion": ""}', NULL, 'kea@agent-kea-ha1');
INSERT INTO public.app VALUES (7, '2025-10-31 09:53:48.247354', 9, 'bind9', true, '{"Version": "BIND 9.20.15 (Stable Release) <id:0c0fcf7>", "ExtendedVersion": ""}', NULL, 'bind9@agent-bind9');
INSERT INTO public.app VALUES (4, '2025-10-31 09:53:46.547234', 3, 'kea', NULL, '{"Version": "3.1.0", "ExtendedVersion": ""}', NULL, 'kea@agent-kea-ha2');
INSERT INTO public.app VALUES (1, '2025-10-31 09:53:44.651209', 4, 'kea', NULL, '{"Version": "3.1.0", "ExtendedVersion": ""}', NULL, 'kea@agent-kea');
INSERT INTO public.app VALUES (8, '2025-10-31 09:53:49.125034', 8, 'bind9', true, '{"Version": "BIND 9.20.15 (Stable Release) <id:0c0fcf7>", "ExtendedVersion": ""}', NULL, 'bind9@agent-bind9-2');


--
-- TOC entry 3914 (class 0 OID 16863)
-- Dependencies: 261
-- Data for Name: bind9_daemon; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.bind9_daemon VALUES (1, 20, '{"ZoneCount": 3, "NamedStats": {"Views": {"guest": {"Zones": null, "Resolver": {"Adb": null, "Cache": null, "Stats": null, "Qtypes": null, "CacheStats": {"CacheHits": 0, "QueryHits": 0, "CacheMisses": 0, "QueryMisses": 0}}}, "trusted": {"Zones": null, "Resolver": {"Adb": null, "Cache": null, "Stats": null, "Qtypes": null, "CacheStats": {"CacheHits": 0, "QueryHits": 0, "CacheMisses": 0, "QueryMisses": 0}}}}, "Memory": null, "Qtypes": null, "Rcodes": null, "NsStats": null, "OpCodes": null, "TaskMgr": null, "Traffic": null, "BootTime": "", "SockStats": null, "SocketMgr": null, "ConfigTime": "", "CurrentTime": "", "NamedVersion": "", "JSONStatsVersion": ""}, "AutomaticZoneCount": 200}');
INSERT INTO public.bind9_daemon VALUES (2, 21, '{"ZoneCount": 6, "NamedStats": {"Views": {"_default": {"Zones": null, "Resolver": {"Adb": null, "Cache": null, "Stats": null, "Qtypes": null, "CacheStats": {"CacheHits": 28, "QueryHits": 0, "CacheMisses": 52, "QueryMisses": 0}}}}, "Memory": null, "Qtypes": null, "Rcodes": null, "NsStats": null, "OpCodes": null, "TaskMgr": null, "Traffic": null, "BootTime": "", "SockStats": null, "SocketMgr": null, "ConfigTime": "", "CurrentTime": "", "NamedVersion": "", "JSONStatsVersion": ""}, "AutomaticZoneCount": 100}');


--
-- TOC entry 3930 (class 0 OID 17047)
-- Dependencies: 277
-- Data for Name: config_checker_preference; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- TOC entry 3923 (class 0 OID 16950)
-- Dependencies: 270
-- Data for Name: config_report; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.config_report VALUES (1, '2025-10-31 09:53:44.702253', 'agent_credentials_over_https', NULL, 1);
INSERT INTO public.config_report VALUES (2, '2025-10-31 09:53:44.702253', 'ca_control_sockets', NULL, 1);
INSERT INTO public.config_report VALUES (3, '2025-10-31 09:53:44.704505', 'lease_cmds_presence', NULL, 3);
INSERT INTO public.config_report VALUES (4, '2025-10-31 09:53:44.704505', 'host_cmds_presence', NULL, 3);
INSERT INTO public.config_report VALUES (5, '2025-10-31 09:53:44.704505', 'dispensable_shared_network', NULL, 3);
INSERT INTO public.config_report VALUES (6, '2025-10-31 09:53:44.704505', 'dispensable_subnet', NULL, 3);
INSERT INTO public.config_report VALUES (7, '2025-10-31 09:53:44.704505', 'out_of_pool_reservation', NULL, 3);
INSERT INTO public.config_report VALUES (8, '2025-10-31 09:53:44.704505', 'overlapping_subnet', 'Kea {daemon} configuration includes 1 overlapping subnet pair. It means that the DHCP clients in different subnets may be assigned the same IP addresses.
1. [16] 192.0.10.0/24 is overlapped by [17] 192.0.10.82/29', 3);
INSERT INTO public.config_report VALUES (9, '2025-10-31 09:53:44.704505', 'canonical_prefix', 'Kea {daemon} configuration contains 1 non-canonical prefix. Kea accepts non-canonical prefix forms, which may lead to duplicates if two subnets have the same prefix specified in different forms. Use canonical forms to ensure that Kea properly identifies and validates subnet prefixes to avoid duplication or overlap.
1. [17] 192.0.10.82/29 is invalid prefix, expected: 192.0.10.80/29', 3);
INSERT INTO public.config_report VALUES (10, '2025-10-31 09:53:44.704505', 'ha_mt_presence', NULL, 3);
INSERT INTO public.config_report VALUES (11, '2025-10-31 09:53:44.704505', 'ha_dedicated_ports', NULL, 3);
INSERT INTO public.config_report VALUES (12, '2025-10-31 09:53:44.704505', 'address_pools_exhausted_by_reservations', NULL, 3);
INSERT INTO public.config_report VALUES (13, '2025-10-31 09:53:44.704505', 'pd_pools_exhausted_by_reservations', NULL, 3);
INSERT INTO public.config_report VALUES (14, '2025-10-31 09:53:44.704505', 'subnet_cmds_and_cb_mutual_exclusion', 'It is recommended that the ''subnet_cmds'' hook library not be used to manage subnets when the configuration backend is used as a source of information about the subnets. The ''subnet_cmds'' hook library modifies the local subnets configuration in the server''s memory, not in the database. Use the ''cb_cmds'' hook library to manage the subnets information in the database instead.', 3);
INSERT INTO public.config_report VALUES (15, '2025-10-31 09:53:44.704505', 'statistics_unavailable_due_to_number_overflow', NULL, 3);
INSERT INTO public.config_report VALUES (16, '2025-10-31 09:53:45.478194', 'lease_cmds_presence', NULL, 5);
INSERT INTO public.config_report VALUES (17, '2025-10-31 09:53:45.478194', 'host_cmds_presence', NULL, 5);
INSERT INTO public.config_report VALUES (18, '2025-10-31 09:53:45.478194', 'dispensable_shared_network', NULL, 5);
INSERT INTO public.config_report VALUES (19, '2025-10-31 09:53:45.478194', 'dispensable_subnet', NULL, 5);
INSERT INTO public.config_report VALUES (20, '2025-10-31 09:53:45.478194', 'out_of_pool_reservation', 'Kea {daemon} configuration includes 1 subnet for which it is recommended to set ''reservations-out-of-pool'' to ''true''. Reservations specified for these subnets appear outside the dynamic-address and/or prefix-delegation pools. Using out-of-pool reservation mode prevents Kea from checking host-reservation existence when allocating in-pool addresses and delegated prefixes, thus improving performance.', 5);
INSERT INTO public.config_report VALUES (21, '2025-10-31 09:53:45.478194', 'overlapping_subnet', NULL, 5);
INSERT INTO public.config_report VALUES (22, '2025-10-31 09:53:45.478194', 'canonical_prefix', NULL, 5);
INSERT INTO public.config_report VALUES (23, '2025-10-31 09:53:45.478194', 'ha_mt_presence', NULL, 5);
INSERT INTO public.config_report VALUES (24, '2025-10-31 09:53:45.478194', 'ha_dedicated_ports', NULL, 5);
INSERT INTO public.config_report VALUES (25, '2025-10-31 09:53:45.478194', 'address_pools_exhausted_by_reservations', NULL, 5);
INSERT INTO public.config_report VALUES (26, '2025-10-31 09:53:45.478194', 'pd_pools_exhausted_by_reservations', NULL, 5);
INSERT INTO public.config_report VALUES (27, '2025-10-31 09:53:45.478194', 'subnet_cmds_and_cb_mutual_exclusion', NULL, 5);
INSERT INTO public.config_report VALUES (28, '2025-10-31 09:53:45.478194', 'statistics_unavailable_due_to_number_overflow', NULL, 5);
INSERT INTO public.config_report VALUES (29, '2025-10-31 09:53:45.48235', 'agent_credentials_over_https', NULL, 6);
INSERT INTO public.config_report VALUES (30, '2025-10-31 09:53:45.48235', 'ca_control_sockets', NULL, 6);
INSERT INTO public.config_report VALUES (31, '2025-10-31 09:53:45.866054', 'agent_credentials_over_https', NULL, 7);
INSERT INTO public.config_report VALUES (32, '2025-10-31 09:53:45.866054', 'ca_control_sockets', NULL, 7);
INSERT INTO public.config_report VALUES (46, '2025-10-31 09:53:46.59009', 'agent_credentials_over_https', NULL, 11);
INSERT INTO public.config_report VALUES (47, '2025-10-31 09:53:46.59009', 'ca_control_sockets', NULL, 11);
INSERT INTO public.config_report VALUES (61, '2025-10-31 09:53:47.063401', 'agent_credentials_over_https', NULL, 17);
INSERT INTO public.config_report VALUES (62, '2025-10-31 09:53:47.063401', 'ca_control_sockets', NULL, 17);
INSERT INTO public.config_report VALUES (76, '2025-10-31 09:53:56.627969', 'lease_cmds_presence', NULL, 9);
INSERT INTO public.config_report VALUES (77, '2025-10-31 09:53:56.627969', 'host_cmds_presence', NULL, 9);
INSERT INTO public.config_report VALUES (78, '2025-10-31 09:53:56.627969', 'dispensable_shared_network', NULL, 9);
INSERT INTO public.config_report VALUES (79, '2025-10-31 09:53:56.627969', 'dispensable_subnet', 'Kea {daemon} configuration includes 1 subnet without pools and host reservations. The DHCP server will not assign any addresses to the devices within this subnet. It is recommended to add some pools or host reservations to this subnet or remove the subnet from the configuration.', 9);
INSERT INTO public.config_report VALUES (80, '2025-10-31 09:53:56.627969', 'out_of_pool_reservation', 'Kea {daemon} configuration includes 1 subnet for which it is recommended to use out-of-pool host-reservation mode. Reservations specified for these subnets are outside the dynamic address pools. Using out-of-pool reservation mode prevents Kea from checking host-reservation existence when allocating in-pool addresses, thus improving performance.', 9);
INSERT INTO public.config_report VALUES (81, '2025-10-31 09:53:56.627969', 'overlapping_subnet', NULL, 9);
INSERT INTO public.config_report VALUES (82, '2025-10-31 09:53:56.627969', 'canonical_prefix', NULL, 9);
INSERT INTO public.config_report VALUES (83, '2025-10-31 09:53:56.627969', 'ha_mt_presence', NULL, 9);
INSERT INTO public.config_report VALUES (84, '2025-10-31 09:53:56.627969', 'ha_dedicated_ports', NULL, 9);
INSERT INTO public.config_report VALUES (85, '2025-10-31 09:53:56.627969', 'address_pools_exhausted_by_reservations', NULL, 9);
INSERT INTO public.config_report VALUES (86, '2025-10-31 09:53:56.627969', 'pd_pools_exhausted_by_reservations', NULL, 9);
INSERT INTO public.config_report VALUES (87, '2025-10-31 09:53:56.627969', 'subnet_cmds_and_cb_mutual_exclusion', NULL, 9);
INSERT INTO public.config_report VALUES (88, '2025-10-31 09:53:56.627969', 'statistics_unavailable_due_to_number_overflow', NULL, 9);
INSERT INTO public.config_report VALUES (89, '2025-10-31 09:53:56.658547', 'lease_cmds_presence', NULL, 15);
INSERT INTO public.config_report VALUES (90, '2025-10-31 09:53:56.658547', 'host_cmds_presence', NULL, 15);
INSERT INTO public.config_report VALUES (91, '2025-10-31 09:53:56.658547', 'dispensable_shared_network', NULL, 15);
INSERT INTO public.config_report VALUES (92, '2025-10-31 09:53:56.658547', 'dispensable_subnet', 'Kea {daemon} configuration includes 1 subnet without pools and host reservations. The DHCP server will not assign any addresses to the devices within this subnet. It is recommended to add some pools or host reservations to this subnet or remove the subnet from the configuration.', 15);
INSERT INTO public.config_report VALUES (93, '2025-10-31 09:53:56.658547', 'out_of_pool_reservation', 'Kea {daemon} configuration includes 1 subnet for which it is recommended to use out-of-pool host-reservation mode. Reservations specified for these subnets are outside the dynamic address pools. Using out-of-pool reservation mode prevents Kea from checking host-reservation existence when allocating in-pool addresses, thus improving performance.', 15);
INSERT INTO public.config_report VALUES (94, '2025-10-31 09:53:56.658547', 'overlapping_subnet', NULL, 15);
INSERT INTO public.config_report VALUES (95, '2025-10-31 09:53:56.658547', 'canonical_prefix', NULL, 15);
INSERT INTO public.config_report VALUES (96, '2025-10-31 09:53:56.658547', 'ha_mt_presence', NULL, 15);
INSERT INTO public.config_report VALUES (97, '2025-10-31 09:53:56.658547', 'ha_dedicated_ports', NULL, 15);
INSERT INTO public.config_report VALUES (98, '2025-10-31 09:53:56.658547', 'address_pools_exhausted_by_reservations', NULL, 15);
INSERT INTO public.config_report VALUES (99, '2025-10-31 09:53:56.658547', 'pd_pools_exhausted_by_reservations', NULL, 15);
INSERT INTO public.config_report VALUES (100, '2025-10-31 09:53:56.658547', 'subnet_cmds_and_cb_mutual_exclusion', NULL, 15);
INSERT INTO public.config_report VALUES (101, '2025-10-31 09:53:56.658547', 'statistics_unavailable_due_to_number_overflow', NULL, 15);
INSERT INTO public.config_report VALUES (102, '2025-10-31 09:53:56.664454', 'lease_cmds_presence', NULL, 13);
INSERT INTO public.config_report VALUES (103, '2025-10-31 09:53:56.664454', 'host_cmds_presence', NULL, 13);
INSERT INTO public.config_report VALUES (104, '2025-10-31 09:53:56.664454', 'dispensable_shared_network', NULL, 13);
INSERT INTO public.config_report VALUES (105, '2025-10-31 09:53:56.664454', 'dispensable_subnet', 'Kea {daemon} configuration includes 1 subnet without pools and host reservations. The DHCP server will not assign any addresses to the devices within this subnet. It is recommended to add some pools or host reservations to this subnet or remove the subnet from the configuration.', 13);
INSERT INTO public.config_report VALUES (106, '2025-10-31 09:53:56.664454', 'out_of_pool_reservation', 'Kea {daemon} configuration includes 1 subnet for which it is recommended to use out-of-pool host-reservation mode. Reservations specified for these subnets are outside the dynamic address pools. Using out-of-pool reservation mode prevents Kea from checking host-reservation existence when allocating in-pool addresses, thus improving performance.', 13);
INSERT INTO public.config_report VALUES (107, '2025-10-31 09:53:56.664454', 'overlapping_subnet', NULL, 13);
INSERT INTO public.config_report VALUES (108, '2025-10-31 09:53:56.664454', 'canonical_prefix', NULL, 13);
INSERT INTO public.config_report VALUES (109, '2025-10-31 09:53:56.664454', 'ha_mt_presence', NULL, 13);
INSERT INTO public.config_report VALUES (110, '2025-10-31 09:53:56.664454', 'ha_dedicated_ports', NULL, 13);
INSERT INTO public.config_report VALUES (111, '2025-10-31 09:53:56.664454', 'address_pools_exhausted_by_reservations', NULL, 13);
INSERT INTO public.config_report VALUES (112, '2025-10-31 09:53:56.664454', 'pd_pools_exhausted_by_reservations', NULL, 13);
INSERT INTO public.config_report VALUES (113, '2025-10-31 09:53:56.664454', 'subnet_cmds_and_cb_mutual_exclusion', NULL, 13);
INSERT INTO public.config_report VALUES (114, '2025-10-31 09:53:56.664454', 'statistics_unavailable_due_to_number_overflow', NULL, 13);


--
-- TOC entry 3926 (class 0 OID 16984)
-- Dependencies: 273
-- Data for Name: config_review; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.config_review VALUES (1, 1, '2025-10-31 09:53:44.702253', 'e3c0c3ba2d8ae9452ed6d8de5ac7b085', '4235041e98940461fb29ea18aeabbf78');
INSERT INTO public.config_review VALUES (2, 3, '2025-10-31 09:53:44.704505', '1d86372a8659455417ca6c94a4f5e71d', '4235041e98940461fb29ea18aeabbf78');
INSERT INTO public.config_review VALUES (3, 5, '2025-10-31 09:53:45.478194', 'daa77bd6683496c08ca1e00e5261cf48', '4235041e98940461fb29ea18aeabbf78');
INSERT INTO public.config_review VALUES (4, 6, '2025-10-31 09:53:45.48235', '2277344267b85d87892c445d00e54dec', '4235041e98940461fb29ea18aeabbf78');
INSERT INTO public.config_review VALUES (5, 7, '2025-10-31 09:53:45.866054', 'e3c0c3ba2d8ae9452ed6d8de5ac7b085', '4235041e98940461fb29ea18aeabbf78');
INSERT INTO public.config_review VALUES (7, 11, '2025-10-31 09:53:46.59009', '3e0079c0f26e59214a32d9b6efda4010', '4235041e98940461fb29ea18aeabbf78');
INSERT INTO public.config_review VALUES (9, 17, '2025-10-31 09:53:47.063401', '5da849ad8ca56e5ca4b424de8f9bcf51', '4235041e98940461fb29ea18aeabbf78');
INSERT INTO public.config_review VALUES (6, 9, '2025-10-31 09:53:56.627969', '4c3624ccc1eaee4be1dac7ed3e85fb1f', '4235041e98940461fb29ea18aeabbf78');
INSERT INTO public.config_review VALUES (10, 15, '2025-10-31 09:53:56.658547', 'e39799467b2a246d9aa6ea243df305d7', '4235041e98940461fb29ea18aeabbf78');
INSERT INTO public.config_review VALUES (8, 13, '2025-10-31 09:53:56.664454', 'd570a4cccf0d2d18b7f219ffc3c0f26e', '4235041e98940461fb29ea18aeabbf78');


--
-- TOC entry 3908 (class 0 OID 16815)
-- Dependencies: 255
-- Data for Name: daemon; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.daemon VALUES (10, 3, NULL, 'dhcp6', false, NULL, NULL, NULL, '2025-10-31 09:53:45.842424', NULL, false);
INSERT INTO public.daemon VALUES (14, 4, NULL, 'dhcp6', false, NULL, NULL, NULL, '2025-10-31 09:53:46.547234', NULL, false);
INSERT INTO public.daemon VALUES (6, 2, NULL, 'ca', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025', NULL, '2025-10-31 09:53:45.426795', NULL, true);
INSERT INTO public.daemon VALUES (4, 1, NULL, 'dhcp6', false, NULL, NULL, NULL, '2025-10-31 09:53:44.651209', NULL, false);
INSERT INTO public.daemon VALUES (5, 2, NULL, 'dhcp6', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025
lease backends:
- Memfile backend 5.0
- PostgreSQL backend 30.0, library 150014
host backends:
- PostgreSQL backend 30.0, library 150014
forensic backends:
- PostgreSQL backend 30.0, library 150014', 697, '2025-10-31 09:53:45.426795', '2025-10-31 09:49:00.786184', true);
INSERT INTO public.daemon VALUES (19, 6, NULL, 'pdns', true, '4.7.3', '4.7.3', 704, '2025-10-31 09:53:47.874309', NULL, true);
INSERT INTO public.daemon VALUES (1, 1, NULL, 'ca', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025', NULL, '2025-10-31 09:53:44.651209', NULL, true);
INSERT INTO public.daemon VALUES (17, 5, NULL, 'ca', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025', NULL, '2025-10-31 09:53:47.037418', NULL, true);
INSERT INTO public.daemon VALUES (2, 1, NULL, 'd2', false, NULL, NULL, NULL, '2025-10-31 09:53:44.651209', NULL, false);
INSERT INTO public.daemon VALUES (3, 1, NULL, 'dhcp4', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025
lease backends:
- Memfile backend 3.0
- MySQL backend 31.0, library 3.3.17
host backends:
- MySQL backend 31.0, library 3.3.17
forensic backends:
- MySQL backend 31.0, library 3.3.17', 696, '2025-10-31 09:53:44.651209', '2025-10-31 09:49:02.9164', true);
INSERT INTO public.daemon VALUES (18, 5, NULL, 'd2', false, NULL, NULL, NULL, '2025-10-31 09:53:47.037418', NULL, false);
INSERT INTO public.daemon VALUES (21, 8, NULL, 'named', true, 'BIND 9.20.15 (Stable Release) <id:0c0fcf7>', NULL, 703, '2025-10-31 09:53:49.125034', '2025-10-31 09:48:54', true);
INSERT INTO public.daemon VALUES (15, 5, NULL, 'dhcp4', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025
lease backends:
- Memfile backend 3.0
- MySQL backend 31.0, library 3.3.17
host backends:
- MySQL backend 31.0, library 3.3.17
forensic backends:
- MySQL backend 31.0, library 3.3.17', 696, '2025-10-31 09:53:47.037418', '2025-10-31 09:49:01.805766', true);
INSERT INTO public.daemon VALUES (16, 5, NULL, 'dhcp6', false, NULL, NULL, NULL, '2025-10-31 09:53:47.037418', NULL, false);
INSERT INTO public.daemon VALUES (20, 7, NULL, 'named', true, 'BIND 9.20.15 (Stable Release) <id:0c0fcf7>', NULL, 703, '2025-10-31 09:53:48.247354', '2025-10-31 09:48:54', true);
INSERT INTO public.daemon VALUES (11, 4, NULL, 'ca', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025', NULL, '2025-10-31 09:53:46.547234', NULL, true);
INSERT INTO public.daemon VALUES (12, 4, NULL, 'd2', false, NULL, NULL, NULL, '2025-10-31 09:53:46.547234', NULL, false);
INSERT INTO public.daemon VALUES (7, 3, NULL, 'ca', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025', NULL, '2025-10-31 09:53:45.842424', NULL, true);
INSERT INTO public.daemon VALUES (8, 3, NULL, 'd2', false, NULL, NULL, NULL, '2025-10-31 09:53:45.842424', NULL, false);
INSERT INTO public.daemon VALUES (13, 4, NULL, 'dhcp4', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025
lease backends:
- Memfile backend 3.0
- MySQL backend 31.0, library 3.3.17
host backends:
- MySQL backend 31.0, library 3.3.17
forensic backends:
- MySQL backend 31.0, library 3.3.17', 696, '2025-10-31 09:53:46.547234', '2025-10-31 09:49:01.898404', true);
INSERT INTO public.daemon VALUES (9, 3, NULL, 'dhcp4', true, '3.1.0', '3.1.0 (3.1.0 (isc20250728104543 deb))
premium: yes (isc20250728104543 deb)
linked with:
- log4cplus 2.0.8
- OpenSSL 3.0.17 1 Jul 2025
lease backends:
- Memfile backend 3.0
- MySQL backend 31.0, library 3.3.17
host backends:
- MySQL backend 31.0, library 3.3.17
forensic backends:
- MySQL backend 31.0, library 3.3.17', 696, '2025-10-31 09:53:45.842424', '2025-10-31 09:49:01.761593', true);


--
-- TOC entry 3924 (class 0 OID 16964)
-- Dependencies: 271
-- Data for Name: daemon_to_config_report; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.daemon_to_config_report VALUES (3, 8, 0);
INSERT INTO public.daemon_to_config_report VALUES (3, 9, 0);
INSERT INTO public.daemon_to_config_report VALUES (3, 14, 0);
INSERT INTO public.daemon_to_config_report VALUES (5, 20, 0);
INSERT INTO public.daemon_to_config_report VALUES (9, 79, 0);
INSERT INTO public.daemon_to_config_report VALUES (9, 80, 0);
INSERT INTO public.daemon_to_config_report VALUES (15, 92, 0);
INSERT INTO public.daemon_to_config_report VALUES (15, 93, 0);
INSERT INTO public.daemon_to_config_report VALUES (13, 105, 0);
INSERT INTO public.daemon_to_config_report VALUES (13, 106, 0);


--
-- TOC entry 3887 (class 0 OID 16568)
-- Dependencies: 234
-- Data for Name: daemon_to_service; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.daemon_to_service VALUES (9, 1);
INSERT INTO public.daemon_to_service VALUES (13, 2);
INSERT INTO public.daemon_to_service VALUES (13, 1);
INSERT INTO public.daemon_to_service VALUES (15, 2);


--
-- TOC entry 3916 (class 0 OID 16898)
-- Dependencies: 263
-- Data for Name: event; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.event VALUES (1, '2025-10-31 09:48:56.402938', 'started Stork Server', 0, '{}', 'version: 2.3.0
build date: 2025-10-31 09:41', '{}');
INSERT INTO public.event VALUES (2, '2025-10-31 09:49:00.233468', 'added <machine id="1" address="agent-kea6" hostname="">', 0, '{"MachineID": 1}', NULL, '{registration}');
INSERT INTO public.event VALUES (3, '2025-10-31 09:49:01.728948', 'added <machine id="2" address="agent-kea-ha3" hostname="">', 0, '{"MachineID": 2}', NULL, '{registration}');
INSERT INTO public.event VALUES (4, '2025-10-31 09:49:01.818364', 'added <machine id="3" address="agent-kea-ha2" hostname="">', 0, '{"MachineID": 3}', NULL, '{registration}');
INSERT INTO public.event VALUES (5, '2025-10-31 09:49:01.938162', 'added <machine id="4" address="agent-kea" hostname="">', 0, '{"MachineID": 4}', NULL, '{registration}');
INSERT INTO public.event VALUES (6, '2025-10-31 09:49:01.965392', 'added <machine id="5" address="agent-kea-ha1" hostname="">', 0, '{"MachineID": 5}', NULL, '{registration}');
INSERT INTO public.event VALUES (7, '2025-10-31 09:49:03.822208', 'added <machine id="6" address="agent-kea-large" hostname="">', 0, '{"MachineID": 6}', NULL, '{registration}');
INSERT INTO public.event VALUES (8, '2025-10-31 09:49:03.841143', 'added <machine id="7" address="agent-pdns" hostname="">', 0, '{"MachineID": 7}', NULL, '{registration}');
INSERT INTO public.event VALUES (9, '2025-10-31 09:49:04.446961', 'added <machine id="8" address="agent-bind9-2" hostname="">', 0, '{"MachineID": 8}', NULL, '{registration}');
INSERT INTO public.event VALUES (10, '2025-10-31 09:49:04.490322', 'added <machine id="9" address="agent-bind9" hostname="">', 0, '{"MachineID": 9}', NULL, '{registration}');
INSERT INTO public.event VALUES (11, '2025-10-31 09:53:44.647697', 'processing Kea command by {daemon} on <machine id="4" address="agent-kea" hostname="agent-kea"> failed', 2, '{"MachineID": 4}', 'unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (12, '2025-10-31 09:53:44.649335', 'processing Kea command by {daemon} on <machine id="4" address="agent-kea" hostname="agent-kea"> failed', 2, '{"MachineID": 4}', 'unable to forward command to the d2 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (13, '2025-10-31 09:53:44.67256', 'added <daemon id="1" name="ca" appId="1" appType="kea"> to <app id="1" name="kea@agent-kea" type="kea" version="3.1.0">', 0, '{"AppID": 1, "DaemonID": 1, "MachineID": 4}', NULL, '{}');
INSERT INTO public.event VALUES (14, '2025-10-31 09:53:44.679718', 'added <daemon id="2" name="d2" appId="1" appType="kea"> to <app id="1" name="kea@agent-kea" type="kea" version="3.1.0">', 0, '{"AppID": 1, "DaemonID": 2, "MachineID": 4}', NULL, '{}');
INSERT INTO public.event VALUES (15, '2025-10-31 09:53:44.680203', 'added <daemon id="3" name="dhcp4" appId="1" appType="kea"> to <app id="1" name="kea@agent-kea" type="kea" version="3.1.0">', 0, '{"AppID": 1, "DaemonID": 3, "MachineID": 4}', NULL, '{}');
INSERT INTO public.event VALUES (16, '2025-10-31 09:53:44.680547', 'added <daemon id="4" name="dhcp6" appId="1" appType="kea"> to <app id="1" name="kea@agent-kea" type="kea" version="3.1.0">', 0, '{"AppID": 1, "DaemonID": 4, "MachineID": 4}', NULL, '{}');
INSERT INTO public.event VALUES (17, '2025-10-31 09:53:44.700522', 'added <subnet id="11" prefix="192.0.2.0/24"> to <daemon id="3" name="dhcp4" appId="1" appType="kea"> in <app id="1" name="kea@agent-kea" type="kea" version="3.1.0">', 0, '{"AppID": 1, "DaemonID": 3, "SubnetID": 11, "MachineID": 4}', NULL, '{}');
INSERT INTO public.event VALUES (18, '2025-10-31 09:53:44.701053', 'added 1 subnets to <daemon id="3" name="dhcp4" appId="1" appType="kea"> in <app id="1" name="kea@agent-kea" type="kea" version="3.1.0">', 0, '{"AppID": 1, "DaemonID": 3, "MachineID": 4}', NULL, '{}');
INSERT INTO public.event VALUES (19, '2025-10-31 09:53:45.457975', 'added <daemon id="5" name="dhcp6" appId="2" appType="kea"> to <app id="2" name="kea@agent-kea6" type="kea" version="3.1.0">', 0, '{"AppID": 2, "DaemonID": 5, "MachineID": 1}', NULL, '{}');
INSERT INTO public.event VALUES (20, '2025-10-31 09:53:45.458777', 'added <daemon id="6" name="ca" appId="2" appType="kea"> to <app id="2" name="kea@agent-kea6" type="kea" version="3.1.0">', 0, '{"AppID": 2, "DaemonID": 6, "MachineID": 1}', NULL, '{}');
INSERT INTO public.event VALUES (21, '2025-10-31 09:53:45.475397', 'added <subnet id="19" prefix="3001:db8:1::/64"> to <daemon id="5" name="dhcp6" appId="2" appType="kea"> in <app id="2" name="kea@agent-kea6" type="kea" version="3.1.0">', 0, '{"AppID": 2, "DaemonID": 5, "SubnetID": 19, "MachineID": 1}', NULL, '{}');
INSERT INTO public.event VALUES (22, '2025-10-31 09:53:45.476085', 'added <subnet id="20" prefix="3000:db8:1::/64"> to <daemon id="5" name="dhcp6" appId="2" appType="kea"> in <app id="2" name="kea@agent-kea6" type="kea" version="3.1.0">', 0, '{"AppID": 2, "DaemonID": 5, "SubnetID": 20, "MachineID": 1}', NULL, '{}');
INSERT INTO public.event VALUES (23, '2025-10-31 09:53:45.476459', 'added <subnet id="21" prefix="3001:1234:5678:90ab:cdef:1f2e:3d4c:5b68/125"> to <daemon id="5" name="dhcp6" appId="2" appType="kea"> in <app id="2" name="kea@agent-kea6" type="kea" version="3.1.0">', 0, '{"AppID": 2, "DaemonID": 5, "SubnetID": 21, "MachineID": 1}', NULL, '{}');
INSERT INTO public.event VALUES (24, '2025-10-31 09:53:45.476847', 'added 3 subnets to <daemon id="5" name="dhcp6" appId="2" appType="kea"> in <app id="2" name="kea@agent-kea6" type="kea" version="3.1.0">', 0, '{"AppID": 2, "DaemonID": 5, "MachineID": 1}', NULL, '{}');
INSERT INTO public.event VALUES (25, '2025-10-31 09:53:45.840805', 'processing Kea command by {daemon} on <machine id="2" address="agent-kea-ha3" hostname="agent-kea-ha3"> failed', 2, '{"MachineID": 2}', 'unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (26, '2025-10-31 09:53:45.841516', 'processing Kea command by {daemon} on <machine id="2" address="agent-kea-ha3" hostname="agent-kea-ha3"> failed', 2, '{"MachineID": 2}', 'unable to forward command to the d2 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (27, '2025-10-31 09:53:45.853766', 'added <daemon id="7" name="ca" appId="3" appType="kea"> to <app id="3" name="kea@agent-kea-ha3" type="kea" version="3.1.0">', 0, '{"AppID": 3, "DaemonID": 7, "MachineID": 2}', NULL, '{}');
INSERT INTO public.event VALUES (28, '2025-10-31 09:53:45.854444', 'added <daemon id="8" name="d2" appId="3" appType="kea"> to <app id="3" name="kea@agent-kea-ha3" type="kea" version="3.1.0">', 0, '{"AppID": 3, "DaemonID": 8, "MachineID": 2}', NULL, '{}');
INSERT INTO public.event VALUES (29, '2025-10-31 09:53:45.854961', 'added <daemon id="9" name="dhcp4" appId="3" appType="kea"> to <app id="3" name="kea@agent-kea-ha3" type="kea" version="3.1.0">', 0, '{"AppID": 3, "DaemonID": 9, "MachineID": 2}', NULL, '{}');
INSERT INTO public.event VALUES (30, '2025-10-31 09:53:45.8554', 'added <daemon id="10" name="dhcp6" appId="3" appType="kea"> to <app id="3" name="kea@agent-kea-ha3" type="kea" version="3.1.0">', 0, '{"AppID": 3, "DaemonID": 10, "MachineID": 2}', NULL, '{}');
INSERT INTO public.event VALUES (31, '2025-10-31 09:53:45.86004', 'added <subnet id="24" prefix="192.0.20.0/24"> to <daemon id="9" name="dhcp4" appId="3" appType="kea"> in <app id="3" name="kea@agent-kea-ha3" type="kea" version="3.1.0">', 0, '{"AppID": 3, "DaemonID": 9, "SubnetID": 24, "MachineID": 2}', NULL, '{}');
INSERT INTO public.event VALUES (32, '2025-10-31 09:53:45.86071', 'added 1 subnets to <daemon id="9" name="dhcp4" appId="3" appType="kea"> in <app id="3" name="kea@agent-kea-ha3" type="kea" version="3.1.0">', 0, '{"AppID": 3, "DaemonID": 9, "MachineID": 2}', NULL, '{}');
INSERT INTO public.event VALUES (47, '2025-10-31 09:53:47.879456', 'added <app id="6" name="pdns@agent-pdns" type="pdns" version="4.7.3">', 0, '{"AppID": 6, "MachineID": 7}', NULL, '{}');
INSERT INTO public.event VALUES (48, '2025-10-31 09:53:48.250322', 'added <app id="7" name="bind9@agent-bind9" type="bind9" version="BIND 9.20.15 (Stable Release) <id:0c0fcf7>">', 0, '{"AppID": 7, "MachineID": 9}', NULL, '{}');
INSERT INTO public.event VALUES (49, '2025-10-31 09:53:49.127128', 'added <app id="8" name="bind9@agent-bind9-2" type="bind9" version="BIND 9.20.15 (Stable Release) <id:0c0fcf7>">', 0, '{"AppID": 8, "MachineID": 8}', NULL, '{}');
INSERT INTO public.event VALUES (33, '2025-10-31 09:53:46.544614', 'processing Kea command by {daemon} on <machine id="3" address="agent-kea-ha2" hostname="agent-kea-ha2"> failed', 2, '{"MachineID": 3}', 'unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (34, '2025-10-31 09:53:46.545834', 'processing Kea command by {daemon} on <machine id="3" address="agent-kea-ha2" hostname="agent-kea-ha2"> failed', 2, '{"MachineID": 3}', 'unable to forward command to the d2 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (43, '2025-10-31 09:53:47.047809', 'added <daemon id="15" name="dhcp4" appId="5" appType="kea"> to <app id="5" name="kea@agent-kea-ha1" type="kea" version="3.1.0">', 0, '{"AppID": 5, "DaemonID": 15, "MachineID": 5}', NULL, '{}');
INSERT INTO public.event VALUES (44, '2025-10-31 09:53:47.0485', 'added <daemon id="16" name="dhcp6" appId="5" appType="kea"> to <app id="5" name="kea@agent-kea-ha1" type="kea" version="3.1.0">', 0, '{"AppID": 5, "DaemonID": 16, "MachineID": 5}', NULL, '{}');
INSERT INTO public.event VALUES (45, '2025-10-31 09:53:47.04895', 'added <daemon id="17" name="ca" appId="5" appType="kea"> to <app id="5" name="kea@agent-kea-ha1" type="kea" version="3.1.0">', 0, '{"AppID": 5, "DaemonID": 17, "MachineID": 5}', NULL, '{}');
INSERT INTO public.event VALUES (46, '2025-10-31 09:53:47.049412', 'added <daemon id="18" name="d2" appId="5" appType="kea"> to <app id="5" name="kea@agent-kea-ha1" type="kea" version="3.1.0">', 0, '{"AppID": 5, "DaemonID": 18, "MachineID": 5}', NULL, '{}');
INSERT INTO public.event VALUES (35, '2025-10-31 09:53:46.571895', 'added <daemon id="11" name="ca" appId="4" appType="kea"> to <app id="4" name="kea@agent-kea-ha2" type="kea" version="3.1.0">', 0, '{"AppID": 4, "DaemonID": 11, "MachineID": 3}', NULL, '{}');
INSERT INTO public.event VALUES (36, '2025-10-31 09:53:46.572868', 'added <daemon id="12" name="d2" appId="4" appType="kea"> to <app id="4" name="kea@agent-kea-ha2" type="kea" version="3.1.0">', 0, '{"AppID": 4, "DaemonID": 12, "MachineID": 3}', NULL, '{}');
INSERT INTO public.event VALUES (37, '2025-10-31 09:53:46.573357', 'added <daemon id="13" name="dhcp4" appId="4" appType="kea"> to <app id="4" name="kea@agent-kea-ha2" type="kea" version="3.1.0">', 0, '{"AppID": 4, "DaemonID": 13, "MachineID": 3}', NULL, '{}');
INSERT INTO public.event VALUES (38, '2025-10-31 09:53:46.573774', 'added <daemon id="14" name="dhcp6" appId="4" appType="kea"> to <app id="4" name="kea@agent-kea-ha2" type="kea" version="3.1.0">', 0, '{"AppID": 4, "DaemonID": 14, "MachineID": 3}', NULL, '{}');
INSERT INTO public.event VALUES (39, '2025-10-31 09:53:46.584184', 'added <subnet id="25" prefix="192.0.3.0/24"> to <daemon id="13" name="dhcp4" appId="4" appType="kea"> in <app id="4" name="kea@agent-kea-ha2" type="kea" version="3.1.0">', 0, '{"AppID": 4, "DaemonID": 13, "SubnetID": 25, "MachineID": 3}', NULL, '{}');
INSERT INTO public.event VALUES (40, '2025-10-31 09:53:46.58465', 'added 1 subnets to <daemon id="13" name="dhcp4" appId="4" appType="kea"> in <app id="4" name="kea@agent-kea-ha2" type="kea" version="3.1.0">', 0, '{"AppID": 4, "DaemonID": 13, "MachineID": 3}', NULL, '{}');
INSERT INTO public.event VALUES (41, '2025-10-31 09:53:47.035171', 'processing Kea command by {daemon} on <machine id="5" address="agent-kea-ha1" hostname="agent-kea-ha1"> failed', 2, '{"MachineID": 5}', 'unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (42, '2025-10-31 09:53:47.035997', 'processing Kea command by {daemon} on <machine id="5" address="agent-kea-ha1" hostname="agent-kea-ha1"> failed', 2, '{"MachineID": 5}', 'unable to forward command to the d2 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (50, '2025-10-31 09:54:04.588555', 'processing Kea command by <daemon id="4" name="dhcp6" appId="1" appType="kea"> on <machine id="4" address="agent-kea" hostname="agent-kea"> failed', 2, '{"AppID": 1, "DaemonID": 4, "MachineID": 4}', 'unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (51, '2025-10-31 09:54:04.590167', 'processing Kea command by <daemon id="2" name="d2" appId="1" appType="kea"> on <machine id="4" address="agent-kea" hostname="agent-kea"> failed', 2, '{"AppID": 1, "DaemonID": 2, "MachineID": 4}', 'unable to forward command to the d2 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (52, '2025-10-31 09:54:04.630852', 'processing Kea command by <daemon id="10" name="dhcp6" appId="3" appType="kea"> on <machine id="2" address="agent-kea-ha3" hostname="agent-kea-ha3"> failed', 2, '{"AppID": 3, "DaemonID": 10, "MachineID": 2}', 'unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (53, '2025-10-31 09:54:04.631393', 'processing Kea command by <daemon id="8" name="d2" appId="3" appType="kea"> on <machine id="2" address="agent-kea-ha3" hostname="agent-kea-ha3"> failed', 2, '{"AppID": 3, "DaemonID": 8, "MachineID": 2}', 'unable to forward command to the d2 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (54, '2025-10-31 09:54:04.647768', 'processing Kea command by <daemon id="14" name="dhcp6" appId="4" appType="kea"> on <machine id="3" address="agent-kea-ha2" hostname="agent-kea-ha2"> failed', 2, '{"AppID": 4, "DaemonID": 14, "MachineID": 3}', 'unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (55, '2025-10-31 09:54:04.648315', 'processing Kea command by <daemon id="12" name="d2" appId="4" appType="kea"> on <machine id="3" address="agent-kea-ha2" hostname="agent-kea-ha2"> failed', 2, '{"AppID": 4, "DaemonID": 12, "MachineID": 3}', 'unable to forward command to the d2 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (56, '2025-10-31 09:54:04.662689', 'processing Kea command by <daemon id="16" name="dhcp6" appId="5" appType="kea"> on <machine id="5" address="agent-kea-ha1" hostname="agent-kea-ha1"> failed', 2, '{"AppID": 5, "DaemonID": 16, "MachineID": 5}', 'unable to forward command to the dhcp6 service: No such file or directory. The server is likely to be offline', '{connectivity}');
INSERT INTO public.event VALUES (57, '2025-10-31 09:54:04.663157', 'processing Kea command by <daemon id="18" name="d2" appId="5" appType="kea"> on <machine id="5" address="agent-kea-ha1" hostname="agent-kea-ha1"> failed', 2, '{"AppID": 5, "DaemonID": 18, "MachineID": 5}', 'unable to forward command to the d2 service: No such file or directory. The server is likely to be offline', '{connectivity}');


--
-- TOC entry 3870 (class 0 OID 16386)
-- Dependencies: 217
-- Data for Name: gopg_migrations; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.gopg_migrations VALUES (1, 1, '2025-10-31 09:48:56.182853+00');
INSERT INTO public.gopg_migrations VALUES (2, 2, '2025-10-31 09:48:56.197958+00');
INSERT INTO public.gopg_migrations VALUES (3, 3, '2025-10-31 09:48:56.219152+00');
INSERT INTO public.gopg_migrations VALUES (4, 4, '2025-10-31 09:48:56.22249+00');
INSERT INTO public.gopg_migrations VALUES (5, 5, '2025-10-31 09:48:56.226147+00');
INSERT INTO public.gopg_migrations VALUES (6, 6, '2025-10-31 09:48:56.233408+00');
INSERT INTO public.gopg_migrations VALUES (7, 7, '2025-10-31 09:48:56.23561+00');
INSERT INTO public.gopg_migrations VALUES (8, 8, '2025-10-31 09:48:56.236152+00');
INSERT INTO public.gopg_migrations VALUES (9, 9, '2025-10-31 09:48:56.246029+00');
INSERT INTO public.gopg_migrations VALUES (10, 10, '2025-10-31 09:48:56.254909+00');
INSERT INTO public.gopg_migrations VALUES (11, 11, '2025-10-31 09:48:56.259738+00');
INSERT INTO public.gopg_migrations VALUES (12, 12, '2025-10-31 09:48:56.260499+00');
INSERT INTO public.gopg_migrations VALUES (13, 13, '2025-10-31 09:48:56.261874+00');
INSERT INTO public.gopg_migrations VALUES (14, 14, '2025-10-31 09:48:56.265237+00');
INSERT INTO public.gopg_migrations VALUES (15, 15, '2025-10-31 09:48:56.271754+00');
INSERT INTO public.gopg_migrations VALUES (16, 16, '2025-10-31 09:48:56.272881+00');
INSERT INTO public.gopg_migrations VALUES (17, 17, '2025-10-31 09:48:56.274533+00');
INSERT INTO public.gopg_migrations VALUES (18, 18, '2025-10-31 09:48:56.282391+00');
INSERT INTO public.gopg_migrations VALUES (19, 19, '2025-10-31 09:48:56.286445+00');
INSERT INTO public.gopg_migrations VALUES (20, 20, '2025-10-31 09:48:56.288276+00');
INSERT INTO public.gopg_migrations VALUES (21, 21, '2025-10-31 09:48:56.290554+00');
INSERT INTO public.gopg_migrations VALUES (22, 22, '2025-10-31 09:48:56.298201+00');
INSERT INTO public.gopg_migrations VALUES (23, 23, '2025-10-31 09:48:56.300701+00');
INSERT INTO public.gopg_migrations VALUES (24, 24, '2025-10-31 09:48:56.301883+00');
INSERT INTO public.gopg_migrations VALUES (25, 25, '2025-10-31 09:48:56.303032+00');
INSERT INTO public.gopg_migrations VALUES (26, 26, '2025-10-31 09:48:56.305323+00');
INSERT INTO public.gopg_migrations VALUES (27, 27, '2025-10-31 09:48:56.307417+00');
INSERT INTO public.gopg_migrations VALUES (28, 28, '2025-10-31 09:48:56.308741+00');
INSERT INTO public.gopg_migrations VALUES (29, 29, '2025-10-31 09:48:56.309457+00');
INSERT INTO public.gopg_migrations VALUES (30, 30, '2025-10-31 09:48:56.309933+00');
INSERT INTO public.gopg_migrations VALUES (31, 31, '2025-10-31 09:48:56.310745+00');
INSERT INTO public.gopg_migrations VALUES (32, 32, '2025-10-31 09:48:56.311209+00');
INSERT INTO public.gopg_migrations VALUES (33, 33, '2025-10-31 09:48:56.313921+00');
INSERT INTO public.gopg_migrations VALUES (34, 34, '2025-10-31 09:48:56.319189+00');
INSERT INTO public.gopg_migrations VALUES (35, 35, '2025-10-31 09:48:56.319777+00');
INSERT INTO public.gopg_migrations VALUES (36, 36, '2025-10-31 09:48:56.322984+00');
INSERT INTO public.gopg_migrations VALUES (37, 37, '2025-10-31 09:48:56.32597+00');
INSERT INTO public.gopg_migrations VALUES (38, 38, '2025-10-31 09:48:56.329596+00');
INSERT INTO public.gopg_migrations VALUES (39, 39, '2025-10-31 09:48:56.331736+00');
INSERT INTO public.gopg_migrations VALUES (40, 40, '2025-10-31 09:48:56.33426+00');
INSERT INTO public.gopg_migrations VALUES (41, 41, '2025-10-31 09:48:56.335148+00');
INSERT INTO public.gopg_migrations VALUES (42, 42, '2025-10-31 09:48:56.337587+00');
INSERT INTO public.gopg_migrations VALUES (43, 43, '2025-10-31 09:48:56.338974+00');
INSERT INTO public.gopg_migrations VALUES (44, 44, '2025-10-31 09:48:56.339567+00');
INSERT INTO public.gopg_migrations VALUES (45, 45, '2025-10-31 09:48:56.342604+00');
INSERT INTO public.gopg_migrations VALUES (46, 46, '2025-10-31 09:48:56.343094+00');
INSERT INTO public.gopg_migrations VALUES (47, 47, '2025-10-31 09:48:56.343563+00');
INSERT INTO public.gopg_migrations VALUES (48, 48, '2025-10-31 09:48:56.344063+00');
INSERT INTO public.gopg_migrations VALUES (49, 49, '2025-10-31 09:48:56.344513+00');
INSERT INTO public.gopg_migrations VALUES (50, 50, '2025-10-31 09:48:56.345005+00');
INSERT INTO public.gopg_migrations VALUES (51, 51, '2025-10-31 09:48:56.346133+00');
INSERT INTO public.gopg_migrations VALUES (52, 52, '2025-10-31 09:48:56.348928+00');
INSERT INTO public.gopg_migrations VALUES (53, 53, '2025-10-31 09:48:56.354293+00');
INSERT INTO public.gopg_migrations VALUES (54, 54, '2025-10-31 09:48:56.359065+00');
INSERT INTO public.gopg_migrations VALUES (55, 55, '2025-10-31 09:48:56.35982+00');
INSERT INTO public.gopg_migrations VALUES (56, 56, '2025-10-31 09:48:56.361015+00');
INSERT INTO public.gopg_migrations VALUES (57, 57, '2025-10-31 09:48:56.361528+00');
INSERT INTO public.gopg_migrations VALUES (58, 58, '2025-10-31 09:48:56.362062+00');
INSERT INTO public.gopg_migrations VALUES (59, 59, '2025-10-31 09:48:56.368706+00');
INSERT INTO public.gopg_migrations VALUES (60, 60, '2025-10-31 09:48:56.369271+00');
INSERT INTO public.gopg_migrations VALUES (61, 61, '2025-10-31 09:48:56.37405+00');
INSERT INTO public.gopg_migrations VALUES (62, 62, '2025-10-31 09:48:56.374539+00');
INSERT INTO public.gopg_migrations VALUES (63, 63, '2025-10-31 09:48:56.380869+00');
INSERT INTO public.gopg_migrations VALUES (64, 64, '2025-10-31 09:48:56.381769+00');
INSERT INTO public.gopg_migrations VALUES (65, 65, '2025-10-31 09:48:56.385717+00');
INSERT INTO public.gopg_migrations VALUES (66, 66, '2025-10-31 09:48:56.38666+00');
INSERT INTO public.gopg_migrations VALUES (67, 67, '2025-10-31 09:48:56.387512+00');
INSERT INTO public.gopg_migrations VALUES (68, 68, '2025-10-31 09:48:56.389872+00');
INSERT INTO public.gopg_migrations VALUES (69, 69, '2025-10-31 09:48:56.392307+00');
INSERT INTO public.gopg_migrations VALUES (70, 70, '2025-10-31 09:48:56.392909+00');


--
-- TOC entry 3886 (class 0 OID 16541)
-- Dependencies: 233
-- Data for Name: ha_service; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.ha_service VALUES (2, 2, 'dhcp4', 'hot-standby', 15, 13, 'hot-standby', 'hot-standby', NULL, '2025-10-31 10:00:26.9976', '2025-10-31 10:00:22.9976', '{server1}', '{}', true, true, NULL, NULL, false, NULL, NULL, NULL, NULL, false, NULL, NULL, NULL, NULL, 'server2');
INSERT INTO public.ha_service VALUES (1, 1, 'dhcp4', 'hot-standby', 9, 13, 'hot-standby', 'hot-standby', NULL, '2025-10-31 10:00:22.990638', '2025-10-31 10:00:26.990638', '{server3}', '{}', true, true, NULL, NULL, false, NULL, NULL, NULL, NULL, false, NULL, NULL, NULL, NULL, 'server3');


--
-- TOC entry 3900 (class 0 OID 16709)
-- Dependencies: 247
-- Data for Name: host; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.host VALUES (1, '2025-10-31 09:53:44.651209', 1);
INSERT INTO public.host VALUES (2, '2025-10-31 09:53:44.651209', 1);
INSERT INTO public.host VALUES (3, '2025-10-31 09:53:44.651209', 1);
INSERT INTO public.host VALUES (4, '2025-10-31 09:53:44.651209', 1);
INSERT INTO public.host VALUES (5, '2025-10-31 09:53:44.651209', 11);
INSERT INTO public.host VALUES (6, '2025-10-31 09:53:44.651209', 11);
INSERT INTO public.host VALUES (7, '2025-10-31 09:53:44.651209', 11);
INSERT INTO public.host VALUES (8, '2025-10-31 09:53:44.651209', 11);
INSERT INTO public.host VALUES (9, '2025-10-31 09:53:44.651209', 11);
INSERT INTO public.host VALUES (10, '2025-10-31 09:53:44.651209', 11);
INSERT INTO public.host VALUES (11, '2025-10-31 09:53:44.651209', 11);
INSERT INTO public.host VALUES (12, '2025-10-31 09:53:44.651209', NULL);
INSERT INTO public.host VALUES (13, '2025-10-31 09:53:44.651209', NULL);
INSERT INTO public.host VALUES (14, '2025-10-31 09:53:45.426795', 19);
INSERT INTO public.host VALUES (15, '2025-10-31 09:53:45.426795', 19);
INSERT INTO public.host VALUES (16, '2025-10-31 09:53:45.426795', 19);
INSERT INTO public.host VALUES (17, '2025-10-31 09:53:45.426795', 19);
INSERT INTO public.host VALUES (18, '2025-10-31 09:53:45.426795', NULL);
INSERT INTO public.host VALUES (19, '2025-10-31 09:53:45.842424', 24);
INSERT INTO public.host VALUES (20, '2025-10-31 09:53:45.842424', 24);
INSERT INTO public.host VALUES (21, '2025-10-31 09:53:45.842424', 24);
INSERT INTO public.host VALUES (22, '2025-10-31 09:53:46.547234', 25);
INSERT INTO public.host VALUES (23, '2025-10-31 09:53:46.547234', 25);
INSERT INTO public.host VALUES (24, '2025-10-31 09:53:46.547234', 25);
INSERT INTO public.host VALUES (26, '2025-10-31 09:53:56.528557', NULL);
INSERT INTO public.host VALUES (27, '2025-10-31 09:53:56.528557', NULL);
INSERT INTO public.host VALUES (28, '2025-10-31 09:53:56.528557', NULL);
INSERT INTO public.host VALUES (25, '2025-10-31 09:53:47.037418', 22);
INSERT INTO public.host VALUES (29, '2025-10-31 09:53:56.528557', 22);
INSERT INTO public.host VALUES (30, '2025-10-31 09:53:56.528557', 22);
INSERT INTO public.host VALUES (31, '2025-10-31 09:53:56.528557', 22);
INSERT INTO public.host VALUES (32, '2025-10-31 09:53:56.528557', 22);
INSERT INTO public.host VALUES (33, '2025-10-31 09:53:56.528557', 22);
INSERT INTO public.host VALUES (34, '2025-10-31 09:53:56.528557', 22);
INSERT INTO public.host VALUES (35, '2025-10-31 09:53:56.528557', 22);
INSERT INTO public.host VALUES (36, '2025-10-31 09:53:56.528557', 22);
INSERT INTO public.host VALUES (37, '2025-10-31 09:53:56.528557', 22);


--
-- TOC entry 3904 (class 0 OID 16752)
-- Dependencies: 251
-- Data for Name: host_identifier; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.host_identifier VALUES (1, '2025-10-31 09:53:44.651209', 'hw-address', '\x010000000000', 1);
INSERT INTO public.host_identifier VALUES (2, '2025-10-31 09:53:44.651209', 'hw-address', '\x010000000001', 2);
INSERT INTO public.host_identifier VALUES (3, '2025-10-31 09:53:44.651209', 'hw-address', '\x010000000002', 3);
INSERT INTO public.host_identifier VALUES (4, '2025-10-31 09:53:44.651209', 'hw-address', '\x010000000003', 4);
INSERT INTO public.host_identifier VALUES (5, '2025-10-31 09:53:44.651209', 'duid', '\x0102030405', 5);
INSERT INTO public.host_identifier VALUES (6, '2025-10-31 09:53:44.651209', 'client-id', '\x010a0b0c0d0e0f', 6);
INSERT INTO public.host_identifier VALUES (7, '2025-10-31 09:53:44.651209', 'client-id', '\x01112233445566', 7);
INSERT INTO public.host_identifier VALUES (8, '2025-10-31 09:53:44.651209', 'client-id', '\x01122334455667', 8);
INSERT INTO public.host_identifier VALUES (9, '2025-10-31 09:53:44.651209', 'hw-address', '\x1a1b1c1d1e1f', 9);
INSERT INTO public.host_identifier VALUES (10, '2025-10-31 09:53:44.651209', 'flex-id', '\x6f75742d6f662d706f6f6c', 10);
INSERT INTO public.host_identifier VALUES (11, '2025-10-31 09:53:44.651209', 'flex-id', '\x73306d4556614c7565', 11);
INSERT INTO public.host_identifier VALUES (12, '2025-10-31 09:53:44.651209', 'client-id', '\xaaaaaaaaaaaa', 12);
INSERT INTO public.host_identifier VALUES (13, '2025-10-31 09:53:44.651209', 'hw-address', '\xeeeeeeeeeeee', 13);
INSERT INTO public.host_identifier VALUES (14, '2025-10-31 09:53:45.426795', 'hw-address', '\x000102030405', 14);
INSERT INTO public.host_identifier VALUES (15, '2025-10-31 09:53:45.426795', 'duid', '\x01020304050a0b0c0d0e', 15);
INSERT INTO public.host_identifier VALUES (16, '2025-10-31 09:53:45.426795', 'duid', '\x300100000000', 16);
INSERT INTO public.host_identifier VALUES (17, '2025-10-31 09:53:45.426795', 'flex-id', '\x736f6d6576616c7565', 17);
INSERT INTO public.host_identifier VALUES (18, '2025-10-31 09:53:45.426795', 'duid', '\x01020304050a0a0a0a0a', 18);
INSERT INTO public.host_identifier VALUES (19, '2025-10-31 09:53:45.842424', 'hw-address', '\x000c01020304', 19);
INSERT INTO public.host_identifier VALUES (20, '2025-10-31 09:53:45.842424', 'hw-address', '\x000c01020305', 20);
INSERT INTO public.host_identifier VALUES (21, '2025-10-31 09:53:45.842424', 'hw-address', '\x000c01020306', 21);
INSERT INTO public.host_identifier VALUES (22, '2025-10-31 09:53:46.547234', 'hw-address', '\x000c01020304', 22);
INSERT INTO public.host_identifier VALUES (23, '2025-10-31 09:53:46.547234', 'hw-address', '\x000c01020305', 23);
INSERT INTO public.host_identifier VALUES (24, '2025-10-31 09:53:46.547234', 'hw-address', '\x000c01020306', 24);
INSERT INTO public.host_identifier VALUES (26, '2025-10-31 09:53:56.528557', 'hw-address', '\x080808080808', 26);
INSERT INTO public.host_identifier VALUES (27, '2025-10-31 09:53:56.528557', 'hw-address', '\x090909090909', 27);
INSERT INTO public.host_identifier VALUES (28, '2025-10-31 09:53:56.528557', 'hw-address', '\x0a0a0a0a0a0a', 28);
INSERT INTO public.host_identifier VALUES (25, '2025-10-31 09:53:47.037418', 'hw-address', '\x010101010101', 25);
INSERT INTO public.host_identifier VALUES (29, '2025-10-31 09:53:56.528557', 'hw-address', '\x020202020202', 29);
INSERT INTO public.host_identifier VALUES (30, '2025-10-31 09:53:56.528557', 'hw-address', '\x030303030303', 30);
INSERT INTO public.host_identifier VALUES (31, '2025-10-31 09:53:56.528557', 'hw-address', '\x040404040404', 31);
INSERT INTO public.host_identifier VALUES (32, '2025-10-31 09:53:56.528557', 'hw-address', '\x050505050505', 32);
INSERT INTO public.host_identifier VALUES (33, '2025-10-31 09:53:56.528557', 'hw-address', '\x060606060606', 33);
INSERT INTO public.host_identifier VALUES (34, '2025-10-31 09:53:56.528557', 'circuit-id', '\x07070707', 34);
INSERT INTO public.host_identifier VALUES (35, '2025-10-31 09:53:56.528557', 'circuit-id', '\x08080808', 35);
INSERT INTO public.host_identifier VALUES (36, '2025-10-31 09:53:56.528557', 'duid', '\x09090909', 36);
INSERT INTO public.host_identifier VALUES (37, '2025-10-31 09:53:56.528557', 'circuit-id', '\x0a0a0a0a', 37);


--
-- TOC entry 3902 (class 0 OID 16723)
-- Dependencies: 249
-- Data for Name: ip_reservation; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.ip_reservation VALUES (1, '2025-10-31 09:53:44.651209', '192.0.5.1/32', 1);
INSERT INTO public.ip_reservation VALUES (2, '2025-10-31 09:53:44.651209', '192.0.5.2/32', 2);
INSERT INTO public.ip_reservation VALUES (3, '2025-10-31 09:53:44.651209', '192.0.5.55/32', 3);
INSERT INTO public.ip_reservation VALUES (4, '2025-10-31 09:53:44.651209', '192.0.5.56/32', 4);
INSERT INTO public.ip_reservation VALUES (5, '2025-10-31 09:53:44.651209', '192.0.2.103/32', 5);
INSERT INTO public.ip_reservation VALUES (6, '2025-10-31 09:53:44.651209', '192.0.2.105/32', 6);
INSERT INTO public.ip_reservation VALUES (7, '2025-10-31 09:53:44.651209', '192.0.2.102/32', 7);
INSERT INTO public.ip_reservation VALUES (8, '2025-10-31 09:53:44.651209', '192.0.2.104/32', 8);
INSERT INTO public.ip_reservation VALUES (9, '2025-10-31 09:53:44.651209', '192.0.2.1/32', 9);
INSERT INTO public.ip_reservation VALUES (10, '2025-10-31 09:53:44.651209', '192.0.2.222/32', 10);
INSERT INTO public.ip_reservation VALUES (11, '2025-10-31 09:53:44.651209', '192.0.2.106/32', 11);
INSERT INTO public.ip_reservation VALUES (12, '2025-10-31 09:53:44.651209', '10.0.0.222/32', 12);
INSERT INTO public.ip_reservation VALUES (13, '2025-10-31 09:53:44.651209', '10.0.0.123/32', 13);
INSERT INTO public.ip_reservation VALUES (14, '2025-10-31 09:53:45.426795', '3001:db8:1::101/128', 14);
INSERT INTO public.ip_reservation VALUES (15, '2025-10-31 09:53:45.426795', '3001:db8:1::100/128', 15);
INSERT INTO public.ip_reservation VALUES (16, '2025-10-31 09:53:45.426795', '3001:db8:1::cafe/128', 16);
INSERT INTO public.ip_reservation VALUES (17, '2025-10-31 09:53:45.426795', '2001:db8:2:abcd::/64', 16);
INSERT INTO public.ip_reservation VALUES (18, '2025-10-31 09:53:45.426795', '3001:db8:1::face/128', 17);
INSERT INTO public.ip_reservation VALUES (19, '2025-10-31 09:53:45.426795', '2001:db8:1::111/128', 18);
INSERT INTO public.ip_reservation VALUES (20, '2025-10-31 09:53:45.426795', '3001:1::/64', 18);
INSERT INTO public.ip_reservation VALUES (21, '2025-10-31 09:53:46.547234', '192.0.20.50/32', 19);
INSERT INTO public.ip_reservation VALUES (24, '2025-10-31 09:53:46.547234', '192.0.20.50/32', 22);
INSERT INTO public.ip_reservation VALUES (22, '2025-10-31 09:53:46.547234', '192.0.20.100/32', 20);
INSERT INTO public.ip_reservation VALUES (25, '2025-10-31 09:53:46.547234', '192.0.20.100/32', 23);
INSERT INTO public.ip_reservation VALUES (23, '2025-10-31 09:53:46.547234', '192.0.20.150/32', 21);
INSERT INTO public.ip_reservation VALUES (26, '2025-10-31 09:53:46.547234', '192.0.20.150/32', 24);
INSERT INTO public.ip_reservation VALUES (27, '2025-10-31 09:53:47.037418', '192.0.3.50/32', 25);
INSERT INTO public.ip_reservation VALUES (31, '2025-10-31 09:53:47.037418', '192.0.3.50/32', 29);
INSERT INTO public.ip_reservation VALUES (28, '2025-10-31 09:53:47.037418', '192.0.3.100/32', 26);
INSERT INTO public.ip_reservation VALUES (32, '2025-10-31 09:53:47.037418', '192.0.3.100/32', 30);
INSERT INTO public.ip_reservation VALUES (29, '2025-10-31 09:53:47.037418', '192.0.3.150/32', 27);
INSERT INTO public.ip_reservation VALUES (33, '2025-10-31 09:53:47.037418', '192.0.3.150/32', 31);
INSERT INTO public.ip_reservation VALUES (37, '2025-10-31 09:53:56.606348', '192.110.111.240/32', 35);
INSERT INTO public.ip_reservation VALUES (53, '2025-10-31 09:53:56.606348', '192.110.111.240/32', 51);
INSERT INTO public.ip_reservation VALUES (69, '2025-10-31 09:53:56.606348', '192.110.111.240/32', 67);
INSERT INTO public.ip_reservation VALUES (38, '2025-10-31 09:53:56.606348', '192.110.111.241/32', 36);
INSERT INTO public.ip_reservation VALUES (54, '2025-10-31 09:53:56.606348', '192.110.111.241/32', 52);
INSERT INTO public.ip_reservation VALUES (70, '2025-10-31 09:53:56.606348', '192.110.111.241/32', 68);
INSERT INTO public.ip_reservation VALUES (39, '2025-10-31 09:53:56.606348', '192.110.111.242/32', 37);
INSERT INTO public.ip_reservation VALUES (55, '2025-10-31 09:53:56.606348', '192.110.111.242/32', 53);
INSERT INTO public.ip_reservation VALUES (71, '2025-10-31 09:53:56.606348', '192.110.111.242/32', 69);
INSERT INTO public.ip_reservation VALUES (30, '2025-10-31 09:53:56.606348', '192.110.111.230/32', 28);
INSERT INTO public.ip_reservation VALUES (40, '2025-10-31 09:53:56.606348', '192.110.111.230/32', 38);
INSERT INTO public.ip_reservation VALUES (56, '2025-10-31 09:53:56.606348', '192.110.111.230/32', 54);
INSERT INTO public.ip_reservation VALUES (72, '2025-10-31 09:53:56.606348', '192.110.111.230/32', 70);
INSERT INTO public.ip_reservation VALUES (41, '2025-10-31 09:53:56.606348', '192.110.111.231/32', 39);
INSERT INTO public.ip_reservation VALUES (57, '2025-10-31 09:53:56.606348', '192.110.111.231/32', 55);
INSERT INTO public.ip_reservation VALUES (73, '2025-10-31 09:53:56.606348', '192.110.111.231/32', 71);
INSERT INTO public.ip_reservation VALUES (42, '2025-10-31 09:53:56.606348', '192.110.111.232/32', 40);
INSERT INTO public.ip_reservation VALUES (58, '2025-10-31 09:53:56.606348', '192.110.111.232/32', 56);
INSERT INTO public.ip_reservation VALUES (74, '2025-10-31 09:53:56.606348', '192.110.111.232/32', 72);
INSERT INTO public.ip_reservation VALUES (43, '2025-10-31 09:53:56.606348', '192.110.111.233/32', 41);
INSERT INTO public.ip_reservation VALUES (59, '2025-10-31 09:53:56.606348', '192.110.111.233/32', 57);
INSERT INTO public.ip_reservation VALUES (75, '2025-10-31 09:53:56.606348', '192.110.111.233/32', 73);
INSERT INTO public.ip_reservation VALUES (44, '2025-10-31 09:53:56.606348', '192.110.111.234/32', 42);
INSERT INTO public.ip_reservation VALUES (60, '2025-10-31 09:53:56.606348', '192.110.111.234/32', 58);
INSERT INTO public.ip_reservation VALUES (76, '2025-10-31 09:53:56.606348', '192.110.111.234/32', 74);
INSERT INTO public.ip_reservation VALUES (45, '2025-10-31 09:53:56.606348', '192.110.111.235/32', 43);
INSERT INTO public.ip_reservation VALUES (61, '2025-10-31 09:53:56.606348', '192.110.111.235/32', 59);
INSERT INTO public.ip_reservation VALUES (77, '2025-10-31 09:53:56.606348', '192.110.111.235/32', 75);
INSERT INTO public.ip_reservation VALUES (46, '2025-10-31 09:53:56.606348', '192.110.111.236/32', 44);
INSERT INTO public.ip_reservation VALUES (62, '2025-10-31 09:53:56.606348', '192.110.111.236/32', 60);
INSERT INTO public.ip_reservation VALUES (78, '2025-10-31 09:53:56.606348', '192.110.111.236/32', 76);
INSERT INTO public.ip_reservation VALUES (47, '2025-10-31 09:53:56.606348', '192.110.111.237/32', 45);
INSERT INTO public.ip_reservation VALUES (63, '2025-10-31 09:53:56.606348', '192.110.111.237/32', 61);
INSERT INTO public.ip_reservation VALUES (79, '2025-10-31 09:53:56.606348', '192.110.111.237/32', 77);
INSERT INTO public.ip_reservation VALUES (48, '2025-10-31 09:53:56.606348', '192.110.111.238/32', 46);
INSERT INTO public.ip_reservation VALUES (64, '2025-10-31 09:53:56.606348', '192.110.111.238/32', 62);
INSERT INTO public.ip_reservation VALUES (80, '2025-10-31 09:53:56.606348', '192.110.111.238/32', 78);
INSERT INTO public.ip_reservation VALUES (49, '2025-10-31 09:53:56.606348', '192.110.111.239/32', 47);
INSERT INTO public.ip_reservation VALUES (65, '2025-10-31 09:53:56.606348', '192.110.111.239/32', 63);
INSERT INTO public.ip_reservation VALUES (81, '2025-10-31 09:53:56.606348', '192.110.111.239/32', 79);


--
-- TOC entry 3910 (class 0 OID 16831)
-- Dependencies: 257
-- Data for Name: kea_daemon; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.kea_daemon VALUES (8, 8, NULL, NULL);
INSERT INTO public.kea_daemon VALUES (9, 9, '{"hash": "9BB04C046F4789BCD6EF0729E01B96C2527139298F62E1F04C4B0DDB6B6240F2", "Dhcp4": {"loggers": [{"name": "kea-dhcp4", "severity": "INFO", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-dhcp4-ha1.log", "maxsize": 10240000, "pattern": ""}]}], "subnet4": [{"id": 1, "pools": [{"pool": "192.0.20.1-192.0.20.200", "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "192.0.20.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [{"code": 3, "data": "192.0.20.1", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 900, "rebind-timer": 1800, "reservations": [{"hostname": "", "hw-address": "00:0c:01:02:03:04", "ip-address": "192.0.20.50", "next-server": "0.0.0.0", "option-data": [{"code": 67, "data": "/tmp/ha-server1/boot.file", "name": "boot-file-name", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 69, "data": "119.12.13.14", "name": "smtp-server", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "00:0c:01:02:03:05", "ip-address": "192.0.20.100", "next-server": "0.0.0.0", "option-data": [{"code": 3, "data": "192.0.20.1", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "00:0c:01:02:03:06", "ip-address": "192.0.20.150", "next-server": "0.0.0.0", "option-data": [{"code": 3, "data": "192.0.20.2", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}], "user-context": {"ha-server-name": "server3"}, "4o6-interface": "", "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}], "allocator": "iterative", "dhcp-ddns": {"sender-ip": "0.0.0.0", "server-ip": "127.0.0.1", "ncr-format": "JSON", "sender-port": 0, "server-port": 53001, "ncr-protocol": "UDP", "enable-updates": false, "max-queue-size": 1024}, "option-def": [], "server-tag": "", "t1-percent": 0.5, "t2-percent": 0.875, "next-server": "0.0.0.0", "option-data": [{"code": 6, "data": "192.0.3.1, 192.0.3.2", "name": "domain-name-servers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 15, "data": "example.org", "name": "domain-name", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 119, "data": "mydomain.example.com, example.com", "name": "domain-search", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 900, "dhcp4o6-port": 0, "rebind-timer": 1800, "authoritative": false, "sanity-checks": {"lease-checks": "warn", "extended-info-checks": "fix"}, "boot-file-name": "", "echo-client-id": true, "lease-database": {"type": "memfile", "lfc-interval": 3600}, "valid-lifetime": 3600, "cache-threshold": 0.25, "control-sockets": [{"socket-name": "/var/run/kea/kea4-ctrl-socket", "socket-type": "unix"}], "hooks-libraries": [{"library": "libdhcp_lease_cmds.so"}, {"library": "libdhcp_host_cmds.so"}, {"library": "libdhcp_subnet_cmds.so"}, {"library": "libdhcp_mysql.so"}, {"library": "libdhcp_ha.so", "parameters": {"high-availability": [{"mode": "hot-standby", "peers": [{"url": "http://172.24.0.121:8005", "name": "server3", "role": "primary", "auto-failover": true}, {"url": "http://172.24.0.110:8006", "name": "server4", "role": "standby", "auto-failover": true}], "sync-leases": true, "sync-timeout": 60000, "max-ack-delay": 5000, "heartbeat-delay": 10000, "multi-threading": {"http-client-threads": 4, "http-listener-threads": 4, "enable-multi-threading": true, "http-dedicated-listener": true}, "sync-page-limit": 10000, "wait-backup-ack": false, "this-server-name": "server3", "restrict-commands": true, "max-response-delay": 20000, "send-lease-updates": true, "max-unacked-clients": 0, "require-client-certs": true, "delayed-updates-limit": 0, "max-rejected-lease-updates": 10}]}}], "hosts-databases": [{"host": "mariadb", "name": "agent_kea_ha3", "type": "mysql", "user": "agent_kea_ha3", "password": "agent_kea_ha3"}], "match-client-id": true, "multi-threading": {"thread-pool-size": 0, "packet-queue-size": 64, "enable-multi-threading": true}, "server-hostname": "", "shared-networks": [{"name": "esperanto", "relay": {"ip-addresses": []}, "subnet4": [{"id": 123, "pools": [], "relay": {"ip-addresses": ["172.103.0.200"]}, "subnet": "192.110.111.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 900, "rebind-timer": 1800, "reservations": [], "user-context": {"site": "esperanto", "subnet-name": "valletta"}, "4o6-interface": "", "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}, {"id": 124, "pools": [], "relay": {"ip-addresses": ["172.103.0.200"]}, "subnet": "192.110.112.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 900, "rebind-timer": 1800, "reservations": [], "user-context": {"site": "esperanto", "subnet-name": "vilnius"}, "4o6-interface": "", "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}], "allocator": "iterative", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 900, "rebind-timer": 1800, "valid-lifetime": 3600, "cache-threshold": 0.25, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}], "ddns-send-updates": true, "hostname-char-set": "[^A-Za-z0-9.-]", "interfaces-config": {"re-detect": true, "interfaces": []}, "dhcp-queue-control": {"capacity": 64, "queue-type": "kea-ring4", "enable-queue": false}, "calculate-tee-times": false, "parked-packet-limit": 256, "reservations-global": false, "stash-agent-options": false, "store-extended-info": false, "ddns-update-on-renew": false, "ddns-generated-prefix": "myhost", "ddns-qualifying-suffix": "", "ip-reservations-unique": true, "reservations-in-subnet": true, "ddns-override-no-update": false, "ddns-replace-client-name": "never", "decline-probation-period": 86400, "reservations-out-of-pool": false, "expired-leases-processing": {"max-reclaim-time": 250, "max-reclaim-leases": 100, "hold-reclaimed-time": 3600, "reclaim-timer-wait-time": 10, "unwarned-reclaim-cycles": 5, "flush-reclaimed-timer-wait-time": 25}, "hostname-char-replacement": "", "reservations-lookup-first": false, "ddns-override-client-update": false, "host-reservation-identifiers": ["hw-address", "duid", "circuit-id", "client-id"], "statistic-default-sample-age": 0, "ddns-conflict-resolution-mode": "check-with-dhcid", "statistic-default-sample-count": 20, "early-global-reservations-lookup": false}}', '4c3624ccc1eaee4be1dac7ed3e85fb1f');
INSERT INTO public.kea_daemon VALUES (2, 2, NULL, NULL);
INSERT INTO public.kea_daemon VALUES (3, 3, '{"hash": "D53DB0D7D28837881720A926F9122FBC3C3BBB77F5ECCE5EEC9768712FEB9E83", "Dhcp4": {"loggers": [{"name": "kea-dhcp4", "severity": "DEBUG", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-dhcp4.log", "maxsize": 10240000, "pattern": ""}]}], "subnet4": [{"id": 1, "pools": [{"pool": "192.0.2.1-192.0.2.50", "option-data": []}, {"pool": "192.0.2.51-192.0.2.100", "option-data": []}, {"pool": "192.0.2.101-192.0.2.150", "option-data": []}, {"pool": "192.0.2.151-192.0.2.200", "option-data": []}], "relay": {"ip-addresses": ["172.100.0.200"]}, "subnet": "192.0.2.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [{"code": 3, "data": "192.0.2.1", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 90, "rebind-timer": 120, "reservations": [{"duid": "01:02:03:04:05", "hostname": "", "ip-address": "192.0.2.103", "next-server": "0.0.0.0", "option-data": [{"code": 6, "data": "10.1.1.202, 10.1.1.203", "name": "domain-name-servers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "client-id": "010A0B0C0D0E0F", "ip-address": "192.0.2.105", "next-server": "192.0.2.1", "option-data": [], "boot-file-name": "/dev/null", "client-classes": [], "server-hostname": "hal9000"}, {"hostname": "special-snowflake", "client-id": "01112233445566", "ip-address": "192.0.2.102", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "client-id": "01122334455667", "ip-address": "192.0.2.104", "next-server": "0.0.0.0", "option-data": [{"code": 125, "data": "4491", "name": "vivso-suboptions", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 2, "data": "10.1.1.202, 10.1.1.203", "name": "tftp-servers", "space": "vendor-4491", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "1a:1b:1c:1d:1e:1f", "ip-address": "192.0.2.1", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"flex-id": "6F75742D6F662D706F6F6C", "hostname": "", "ip-address": "192.0.2.222", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"flex-id": "73306D4556614C7565", "hostname": "", "ip-address": "192.0.2.106", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}], "4o6-interface": "", "client-classes": ["class-00-00"], "valid-lifetime": 180, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 180, "min-valid-lifetime": 180, "calculate-tee-times": false, "store-extended-info": false}], "allocator": "iterative", "dhcp-ddns": {"sender-ip": "0.0.0.0", "server-ip": "127.0.0.1", "ncr-format": "JSON", "sender-port": 0, "server-port": 53001, "ncr-protocol": "UDP", "enable-updates": false, "max-queue-size": 1024}, "option-def": [], "server-tag": "", "t1-percent": 0.5, "t2-percent": 0.875, "next-server": "0.0.0.0", "option-data": [{"code": 6, "data": "192.0.2.1, 192.0.2.2", "name": "domain-name-servers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 15, "data": "example.org", "name": "domain-name", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 119, "data": "mydomain.example.com, example.com", "name": "domain-search", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 67, "data": "EST5EDT4\\,M3.2.0/02:00\\,M11.1.0/02:00", "name": "boot-file-name", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 23, "data": "0xf0", "name": "default-ip-ttl", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 90, "dhcp4o6-port": 0, "rebind-timer": 120, "reservations": [{"hostname": "", "client-id": "AAAAAAAAAAAA", "ip-address": "10.0.0.222", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "ee:ee:ee:ee:ee:ee", "ip-address": "10.0.0.123", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}], "authoritative": false, "sanity-checks": {"lease-checks": "warn", "extended-info-checks": "fix"}, "boot-file-name": "", "client-classes": [{"name": "class-00-00", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''00:00''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-01-00", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''01:00''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-01-01", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''01:01''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-01-02", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''01:02''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-01-03", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''01:03''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-01-04", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''01:04''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-01-05", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''01:05''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-01-06", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''01:06''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-02-00", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''02:00''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-02-01", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''02:01''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}, {"name": "class-02-02", "test": "substring(hexstring(pkt4.mac,'':''),0,5) == ''02:02''", "option-def": [], "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "server-hostname": ""}], "config-control": {"config-databases": [{"host": "mariadb", "name": "agent_kea", "type": "mysql", "user": "agent_kea", "password": "agent_kea"}], "config-fetch-wait-time": 20}, "echo-client-id": true, "lease-database": {"host": "mariadb", "name": "agent_kea", "type": "mysql", "user": "agent_kea", "password": "agent_kea"}, "valid-lifetime": 180, "cache-threshold": 0.25, "control-sockets": [{"socket-name": "/var/run/kea/kea4-ctrl-socket", "socket-type": "unix"}], "hooks-libraries": [{"library": "libdhcp_lease_cmds.so"}, {"library": "libdhcp_stat_cmds.so"}, {"library": "libdhcp_mysql.so"}, {"library": "libdhcp_legal_log.so", "parameters": {"path": "/var/log/kea", "base-name": "kea-legal-log"}}, {"library": "libdhcp_subnet_cmds.so"}], "match-client-id": true, "multi-threading": {"thread-pool-size": 0, "packet-queue-size": 64, "enable-multi-threading": true}, "server-hostname": "", "shared-networks": [{"name": "frog", "relay": {"ip-addresses": ["172.101.0.200"]}, "subnet4": [{"id": 11, "pools": [{"pool": "192.0.5.1-192.0.5.50", "option-data": []}], "relay": {"ip-addresses": ["172.101.0.200"]}, "subnet": "192.0.5.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [{"hostname": "", "hw-address": "01:00:00:00:00:00", "ip-address": "192.0.5.1", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "01:00:00:00:00:01", "ip-address": "192.0.5.2", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "01:00:00:00:00:02", "ip-address": "192.0.5.55", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "01:00:00:00:00:03", "ip-address": "192.0.5.56", "next-server": "0.0.0.0", "option-data": [], "boot-file-name": "", "client-classes": [], "server-hostname": ""}], "user-context": {"baz": 42, "boz": ["a", "b", "c"], "foo": "bar"}, "4o6-interface": "", "client-classes": ["class-01-00"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}, {"id": 12, "pools": [{"pool": "192.0.6.1-192.0.6.40", "pool-id": 6001, "option-data": []}, {"pool": "192.0.6.61-192.0.6.90", "pool-id": 6061, "option-data": []}, {"pool": "192.0.6.111-192.0.6.150", "pool-id": 6111, "option-data": []}], "relay": {"ip-addresses": ["172.101.0.200"]}, "subnet": "192.0.6.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [], "user-context": {}, "4o6-interface": "", "client-classes": ["class-01-01"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}, {"id": 13, "pools": [{"pool": "192.0.7.1-192.0.7.50", "pool-id": 7001, "option-data": []}, {"pool": "192.0.7.51-192.0.7.100", "pool-id": 7051, "option-data": []}], "relay": {"ip-addresses": ["172.101.0.200"]}, "subnet": "192.0.7.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [], "user-context": {"subnet-name": "alice"}, "4o6-interface": "", "client-classes": ["class-01-02"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}, {"id": 14, "pools": [{"pool": "192.0.8.1-192.0.8.50", "pool-id": 8001, "option-data": []}], "relay": {"ip-addresses": ["172.101.0.200"]}, "subnet": "192.0.8.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [], "user-context": {"subnet-name": "bob"}, "4o6-interface": "", "client-classes": ["class-01-03"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}, {"id": 15, "pools": [{"pool": "192.0.9.1-192.0.9.50", "pool-id": 9001, "option-data": []}], "relay": {"ip-addresses": ["172.101.0.200"]}, "subnet": "192.0.9.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [], "4o6-interface": "", "client-classes": ["class-01-04"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}, {"id": 16, "pools": [{"pool": "192.0.10.1-192.0.10.50", "pool-id": 10001, "option-data": []}], "relay": {"ip-addresses": ["172.101.0.200"]}, "subnet": "192.0.10.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [], "4o6-interface": "", "client-classes": ["class-01-05"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}, {"id": 17, "pools": [{"pool": "192.0.10.82/32", "pool-id": 10082, "option-data": []}, {"pool": "192.0.10.83/32", "pool-id": 10082, "option-data": []}, {"pool": "192.0.10.84/32", "pool-id": 10082, "option-data": []}], "relay": {"ip-addresses": ["172.101.0.200"]}, "subnet": "192.0.10.82/29", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [], "4o6-interface": "", "client-classes": ["class-01-06"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}], "allocator": "iterative", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "valid-lifetime": 200, "cache-threshold": 0.25, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}, {"name": "mouse", "relay": {"ip-addresses": ["172.102.0.200"]}, "subnet4": [{"id": 21, "pools": [{"pool": "192.1.15.1-192.1.15.50", "pool-id": 1015001, "option-data": []}], "relay": {"ip-addresses": ["172.102.0.200"]}, "subnet": "192.1.15.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [], "4o6-interface": "", "client-classes": ["class-02-00"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}, {"id": 22, "pools": [{"pool": "192.1.16.1-192.1.16.50", "pool-id": 1016001, "option-data": []}, {"pool": "192.1.16.51-192.1.16.100", "pool-id": 1016051, "option-data": []}, {"pool": "192.1.16.101-192.1.16.150", "pool-id": 1016101, "option-data": []}], "relay": {"ip-addresses": ["172.102.0.200"]}, "subnet": "192.1.16.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [], "4o6-interface": "", "client-classes": ["class-02-01"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}, {"id": 23, "pools": [{"pool": "192.1.17.1-192.1.17.20", "pool-id": 1017001, "option-data": []}, {"pool": "192.1.17.21-192.1.17.40", "pool-id": 1017021, "option-data": []}, {"pool": "192.1.17.41-192.1.17.60", "pool-id": 1017041, "option-data": []}, {"pool": "192.1.17.66-192.1.17.80", "pool-id": 1017066, "option-data": []}, {"pool": "192.1.17.81-192.1.17.100", "pool-id": 1017081, "option-data": []}, {"pool": "192.1.17.101-192.1.17.120", "pool-id": 1017101, "option-data": []}, {"pool": "192.1.17.121-192.1.17.140", "pool-id": 1017121, "option-data": []}, {"pool": "192.1.17.141-192.1.17.160", "pool-id": 1017141, "option-data": []}, {"pool": "192.1.17.161-192.1.17.180", "pool-id": 1017161, "option-data": []}, {"pool": "192.1.17.181-192.1.17.200", "pool-id": 1017181, "option-data": []}, {"pool": "192.1.17.201-192.1.17.220", "pool-id": 1017181, "option-data": []}, {"pool": "192.1.17.221-192.1.17.240", "pool-id": 1017181, "option-data": []}, {"pool": "192.1.17.241-192.1.17.243", "pool-id": 1017241, "option-data": []}, {"pool": "192.1.17.244-192.1.17.246", "pool-id": 1017241, "option-data": []}, {"pool": "192.1.17.247-192.1.17.250", "pool-id": 1017241, "option-data": []}], "relay": {"ip-addresses": ["172.102.0.200"]}, "subnet": "192.1.17.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "reservations": [], "4o6-interface": "", "client-classes": ["class-02-02"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}], "allocator": "iterative", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 90, "rebind-timer": 120, "valid-lifetime": 200, "cache-threshold": 0.25, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false, "store-extended-info": false}], "ddns-send-updates": true, "hostname-char-set": "[^A-Za-z0-9.-]", "interfaces-config": {"re-detect": true, "interfaces": ["*"]}, "dhcp-queue-control": {"capacity": 64, "queue-type": "kea-ring4", "enable-queue": false}, "calculate-tee-times": false, "parked-packet-limit": 256, "reservations-global": false, "stash-agent-options": false, "store-extended-info": false, "ddns-update-on-renew": false, "ddns-generated-prefix": "myhost", "ddns-qualifying-suffix": "", "ip-reservations-unique": true, "reservations-in-subnet": true, "ddns-override-no-update": false, "ddns-replace-client-name": "never", "decline-probation-period": 86400, "reservations-out-of-pool": false, "expired-leases-processing": {"max-reclaim-time": 250, "max-reclaim-leases": 100, "hold-reclaimed-time": 3600, "reclaim-timer-wait-time": 10, "unwarned-reclaim-cycles": 5, "flush-reclaimed-timer-wait-time": 25}, "hostname-char-replacement": "", "reservations-lookup-first": false, "ddns-override-client-update": false, "host-reservation-identifiers": ["hw-address", "duid", "circuit-id", "client-id"], "statistic-default-sample-age": 0, "ddns-conflict-resolution-mode": "check-with-dhcid", "statistic-default-sample-count": 20, "early-global-reservations-lookup": false}}', '1d86372a8659455417ca6c94a4f5e71d');
INSERT INTO public.kea_daemon VALUES (10, 10, NULL, NULL);
INSERT INTO public.kea_daemon VALUES (6, 6, '{"hash": "6DE69059D21F8F094BB6A85B85B21B8FC7B6E7FC6640BBD5BB45942AB940B946", "Control-agent": {"loggers": [{"name": "kea-ctrl-agent", "severity": "INFO", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-ctrl-agent.log", "maxsize": 10240000, "pattern": ""}]}], "http-host": "0.0.0.0", "http-port": 8000, "control-sockets": {"dhcp6": {"socket-name": "/var/run/kea/kea6-ctrl-socket", "socket-type": "unix"}}, "hooks-libraries": []}}', '2277344267b85d87892c445d00e54dec');
INSERT INTO public.kea_daemon VALUES (5, 5, '{"hash": "7A9A213B163597F6CC2A617666CB1D09235B301B7CDA6A0A45BB73E27F7EDEB4", "Dhcp6": {"loggers": [{"name": "kea-dhcp6", "severity": "INFO", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-dhcp6.log", "maxsize": 10240000, "pattern": ""}]}], "subnet6": [{"id": 1, "pools": [{"pool": "3001:db8:1:0:1::/80", "option-data": []}, {"pool": "3001:db8:1:0:2::/80", "option-data": []}, {"pool": "3001:db8:1:0:3::/80", "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "3001:db8:1::/64", "pd-pools": [{"prefix": "3001:db8:8::", "pool-id": 8, "prefix-len": 56, "option-data": [], "delegated-len": 64}, {"prefix": "3001:db8:9::", "pool-id": 9, "prefix-len": 56, "option-data": [], "delegated-len": 64, "excluded-prefix": "3001:db8:9:0:ca00::", "excluded-prefix-len": 72}], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [{"code": 23, "data": "3001:db8:2::dead:beef, 3001:db8:2::cafe:babe", "name": "dns-servers", "space": "dhcp6", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 1000, "pd-allocator": "iterative", "rapid-commit": false, "rebind-timer": 2000, "reservations": [{"hostname": "", "prefixes": [], "hw-address": "00:01:02:03:04:05", "option-data": [{"code": 23, "data": "3000:1::234", "name": "dns-servers", "space": "dhcp6", "csv-format": true, "never-send": false, "always-send": false}, {"code": 27, "data": "3000:1::234", "name": "nis-servers", "space": "dhcp6", "csv-format": true, "never-send": false, "always-send": false}], "ip-addresses": ["3001:db8:1::101"], "client-classes": ["special_snowflake", "office"]}, {"duid": "01:02:03:04:05:0a:0b:0c:0d:0e", "hostname": "", "prefixes": [], "option-data": [], "ip-addresses": ["3001:db8:1::100"], "client-classes": []}, {"duid": "30:01:00:00:00:00", "hostname": "foo.example.com", "prefixes": ["2001:db8:2:abcd::/64"], "option-data": [{"code": 17, "data": "4491", "name": "vendor-opts", "space": "dhcp6", "csv-format": true, "never-send": false, "always-send": false}, {"code": 32, "data": "3000:1::234", "name": "tftp-servers", "space": "vendor-4491", "csv-format": true, "never-send": false, "always-send": false}], "ip-addresses": ["3001:db8:1::cafe"], "client-classes": []}, {"flex-id": "736F6D6576616C7565", "hostname": "", "prefixes": [], "option-data": [], "ip-addresses": ["3001:db8:1::face"], "client-classes": []}], "client-classes": ["class-30-01"], "valid-lifetime": 4000, "cache-threshold": 0.25, "max-valid-lifetime": 4000, "min-valid-lifetime": 4000, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}, {"id": 2, "pools": [{"pool": "3000:db8:1::/80", "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "3000:db8:1::/64", "pd-pools": [], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rapid-commit": false, "rebind-timer": 2000, "reservations": [], "client-classes": ["class-30-00"], "valid-lifetime": 4000, "cache-threshold": 0.25, "max-valid-lifetime": 4000, "min-valid-lifetime": 4000, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}, {"id": 10, "pools": [{"pool": "3001:1234:5678:90ab:cdef:1f2e:3d4c:5b68/126", "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "3001:1234:5678:90ab:cdef:1f2e:3d4c:5b68/125", "pd-pools": [], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rapid-commit": false, "rebind-timer": 2000, "reservations": [], "client-classes": ["class-30-01"], "valid-lifetime": 4000, "cache-threshold": 0.25, "max-valid-lifetime": 4000, "min-valid-lifetime": 4000, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}], "allocator": "iterative", "dhcp-ddns": {"sender-ip": "0.0.0.0", "server-ip": "127.0.0.1", "ncr-format": "JSON", "sender-port": 0, "server-port": 53001, "ncr-protocol": "UDP", "enable-updates": false, "max-queue-size": 1024}, "server-id": {"time": 0, "type": "LLT", "htype": 0, "persist": true, "identifier": "", "enterprise-id": 0}, "option-def": [], "server-tag": "", "t1-percent": 0.5, "t2-percent": 0.8, "mac-sources": ["any"], "option-data": [{"code": 23, "data": "2001:db8:2::45, 2001:db8:2::100", "name": "dns-servers", "space": "dhcp6", "csv-format": true, "never-send": false, "always-send": false}, {"code": 12, "data": "2001:db8::1", "name": "unicast", "space": "dhcp6", "csv-format": true, "never-send": false, "always-send": false}, {"code": 41, "data": "EST5EDT4\\,M3.2.0/02:00\\,M11.1.0/02:00", "name": "new-posix-timezone", "space": "dhcp6", "csv-format": true, "never-send": false, "always-send": false}, {"code": 7, "data": "0xf0", "name": "preference", "space": "dhcp6", "csv-format": true, "never-send": false, "always-send": false}, {"code": 60, "data": "root=/dev/sda2, quiet, splash", "name": "bootfile-param", "space": "dhcp6", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 1000, "dhcp4o6-port": 0, "pd-allocator": "iterative", "rebind-timer": 2000, "reservations": [{"duid": "01:02:03:04:05:0a:0a:0a:0a:0a", "hostname": "", "prefixes": ["3001:1::/64"], "option-data": [], "ip-addresses": ["2001:db8:1::111"], "client-classes": []}], "sanity-checks": {"lease-checks": "warn", "extended-info-checks": "fix"}, "client-classes": [{"name": "class-30-01", "test": "substring(option[1].hex,0,2) == 0x3001", "option-data": []}, {"name": "class-30-00", "test": "substring(option[1].hex,0,2) == 0x3000", "option-data": []}, {"name": "class-40-01", "test": "substring(option[1].hex,0,2) == 0x4001", "option-data": []}, {"name": "class-50-00", "test": "substring(option[1].hex,0,2) == 0x5000", "option-data": []}, {"name": "class-50-01", "test": "substring(option[1].hex,0,2) == 0x5001", "option-data": []}, {"name": "class-50-02", "test": "substring(option[1].hex,0,2) == 0x5002", "option-data": []}, {"name": "class-50-03", "test": "substring(option[1].hex,0,2) == 0x5003", "option-data": []}, {"name": "class-50-04", "test": "substring(option[1].hex,0,2) == 0x5004", "option-data": []}, {"name": "class-50-05", "test": "substring(option[1].hex,0,2) == 0x5005", "option-data": []}], "lease-database": {"host": "postgres", "name": "agent_kea6", "type": "postgresql", "user": "agent_kea6", "password": "agent_kea6"}, "valid-lifetime": 4000, "cache-threshold": 0.25, "control-sockets": [{"socket-name": "/var/run/kea/kea6-ctrl-socket", "socket-type": "unix"}], "hooks-libraries": [{"library": "libdhcp_lease_cmds.so"}, {"library": "libdhcp_pgsql.so"}, {"library": "libdhcp_legal_log.so", "parameters": {"path": "/var/log/kea", "base-name": "kea-legal-log"}}, {"library": "libdhcp_subnet_cmds.so"}], "multi-threading": {"thread-pool-size": 0, "packet-queue-size": 64, "enable-multi-threading": true}, "shared-networks": [{"name": "frog", "relay": {"ip-addresses": []}, "subnet6": [{"id": 3, "pools": [{"pool": "4001:db8:1:0:abcd::/80", "pool-id": 301, "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "4001:db8:1::/64", "pd-pools": [], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rebind-timer": 2000, "reservations": [], "client-classes": ["class-40-01"], "valid-lifetime": 300, "cache-threshold": 0.25, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}, {"id": 4, "pools": [{"pool": "5000:db8::/64", "pool-id": 401, "option-data": []}, {"pool": "5000:dba::/64", "pool-id": 402, "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "5000::/16", "pd-pools": [], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rebind-timer": 2000, "reservations": [], "client-classes": ["class-50-00"], "valid-lifetime": 300, "cache-threshold": 0.25, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}, {"id": 5, "pools": [{"pool": "5001:db8::/64", "pool-id": 501, "option-data": []}, {"pool": "5001:dba::/64", "pool-id": 502, "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "5001::/16", "pd-pools": [], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rebind-timer": 2000, "reservations": [], "client-classes": ["class-50-01"], "valid-lifetime": 300, "cache-threshold": 0.25, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}, {"id": 6, "pools": [{"pool": "5002:db8::/64", "pool-id": 601, "option-data": []}, {"pool": "5002:dba::/64", "pool-id": 602, "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "5002::/16", "pd-pools": [], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rebind-timer": 2000, "reservations": [], "client-classes": ["class-50-02"], "valid-lifetime": 300, "cache-threshold": 0.25, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}, {"id": 7, "pools": [{"pool": "5003:db8::/64", "pool-id": 701, "option-data": []}, {"pool": "5003:dba::/64", "pool-id": 702, "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "5003::/16", "pd-pools": [], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rebind-timer": 2000, "reservations": [], "client-classes": ["class-50-03"], "valid-lifetime": 300, "cache-threshold": 0.25, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}, {"id": 8, "pools": [{"pool": "5004:db8::/64", "pool-id": 801, "option-data": []}, {"pool": "5004:dba::/64", "pool-id": 802, "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "5004::/16", "pd-pools": [], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rebind-timer": 2000, "reservations": [], "client-classes": ["class-50-04"], "valid-lifetime": 300, "cache-threshold": 0.25, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}, {"id": 9, "pools": [{"pool": "5005:db8::/64", "pool-id": 901, "option-data": []}, {"pool": "5005:dba::/64", "pool-id": 902, "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "5005::/16", "pd-pools": [], "allocator": "iterative", "interface": "eth1", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rebind-timer": 2000, "reservations": [], "client-classes": ["class-50-05"], "valid-lifetime": 300, "cache-threshold": 0.25, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}], "allocator": "iterative", "t1-percent": 0.5, "t2-percent": 0.8, "option-data": [], "renew-timer": 1000, "pd-allocator": "iterative", "rapid-commit": false, "rebind-timer": 2000, "valid-lifetime": 300, "cache-threshold": 0.25, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "store-extended-info": false, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}], "ddns-send-updates": true, "hostname-char-set": "[^A-Za-z0-9.-]", "interfaces-config": {"re-detect": true, "interfaces": ["eth1", "eth2"]}, "dhcp-queue-control": {"capacity": 64, "queue-type": "kea-ring6", "enable-queue": false}, "preferred-lifetime": 3000, "calculate-tee-times": true, "parked-packet-limit": 256, "reservations-global": false, "store-extended-info": false, "ddns-update-on-renew": false, "ddns-generated-prefix": "myhost", "ddns-qualifying-suffix": "", "ip-reservations-unique": true, "relay-supplied-options": ["65"], "reservations-in-subnet": true, "ddns-override-no-update": false, "ddns-replace-client-name": "never", "decline-probation-period": 86400, "reservations-out-of-pool": false, "expired-leases-processing": {"max-reclaim-time": 250, "max-reclaim-leases": 100, "hold-reclaimed-time": 3600, "reclaim-timer-wait-time": 10, "unwarned-reclaim-cycles": 5, "flush-reclaimed-timer-wait-time": 25}, "hostname-char-replacement": "", "reservations-lookup-first": false, "ddns-override-client-update": false, "host-reservation-identifiers": ["hw-address", "duid"], "statistic-default-sample-age": 0, "ddns-conflict-resolution-mode": "check-with-dhcid", "statistic-default-sample-count": 20, "early-global-reservations-lookup": false}}', 'daa77bd6683496c08ca1e00e5261cf48');
INSERT INTO public.kea_daemon VALUES (16, 16, NULL, NULL);
INSERT INTO public.kea_daemon VALUES (11, 11, '{"hash": "6DA38C01063CAA712B1E1B88EB7602201A5712CD549B593D333BDF2539404947", "Control-agent": {"loggers": [{"name": "kea-ctrl-agent", "severity": "INFO", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-ctrl-agent.log", "maxsize": 10240000, "pattern": ""}]}], "http-host": "0.0.0.0", "http-port": 8002, "control-sockets": {"d2": {"socket-name": "/var/run/kea/kea-ddns-ctrl-socket", "socket-type": "unix"}, "dhcp4": {"socket-name": "/var/run/kea/kea4-ctrl-socket", "socket-type": "unix"}, "dhcp6": {"socket-name": "/var/run/kea/kea6-ctrl-socket", "socket-type": "unix"}}, "hooks-libraries": []}}', '3e0079c0f26e59214a32d9b6efda4010');
INSERT INTO public.kea_daemon VALUES (17, 17, '{"hash": "7F0B5E168607A8564695BF8B1DF30AB91A7E7B42964D98A7C41E79F91238B9B6", "Control-agent": {"loggers": [{"name": "kea-ctrl-agent", "severity": "INFO", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-ctrl-agent.log", "maxsize": 10240000, "pattern": ""}]}], "http-host": "0.0.0.0", "http-port": 8001, "control-sockets": {"d2": {"socket-name": "/var/run/kea/kea-ddns-ctrl-socket", "socket-type": "unix"}, "dhcp4": {"socket-name": "/var/run/kea/kea4-ctrl-socket", "socket-type": "unix"}, "dhcp6": {"socket-name": "/var/run/kea/kea6-ctrl-socket", "socket-type": "unix"}}, "hooks-libraries": []}}', '5da849ad8ca56e5ca4b424de8f9bcf51');
INSERT INTO public.kea_daemon VALUES (12, 12, NULL, NULL);
INSERT INTO public.kea_daemon VALUES (18, 18, NULL, NULL);
INSERT INTO public.kea_daemon VALUES (15, 15, '{"hash": "BE729684B39E4324303B25BF64A92A4E0E486ED343C2807FCADE5B2B31CEC529", "Dhcp4": {"loggers": [{"name": "kea-dhcp4", "severity": "INFO", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-dhcp4-ha1.log", "maxsize": 10240000, "pattern": ""}]}], "subnet4": [{"id": 1, "pools": [{"pool": "192.0.3.1-192.0.3.200", "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "192.0.3.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [{"code": 3, "data": "192.0.3.1", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 900, "rebind-timer": 1800, "reservations": [{"hostname": "", "hw-address": "00:0c:01:02:03:04", "ip-address": "192.0.3.50", "next-server": "0.0.0.0", "option-data": [{"code": 67, "data": "/tmp/ha-server1/boot.file", "name": "boot-file-name", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 69, "data": "119.12.13.14", "name": "smtp-server", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "00:0c:01:02:03:05", "ip-address": "192.0.3.100", "next-server": "0.0.0.0", "option-data": [{"code": 3, "data": "192.0.3.1", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "00:0c:01:02:03:06", "ip-address": "192.0.3.150", "next-server": "0.0.0.0", "option-data": [{"code": 3, "data": "192.0.3.2", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}], "4o6-interface": "", "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}], "allocator": "iterative", "dhcp-ddns": {"sender-ip": "0.0.0.0", "server-ip": "127.0.0.1", "ncr-format": "JSON", "sender-port": 0, "server-port": 53001, "ncr-protocol": "UDP", "enable-updates": false, "max-queue-size": 1024}, "option-def": [], "server-tag": "", "t1-percent": 0.5, "t2-percent": 0.875, "next-server": "0.0.0.0", "option-data": [{"code": 6, "data": "192.0.3.1, 192.0.3.2", "name": "domain-name-servers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 15, "data": "example.org", "name": "domain-name", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 119, "data": "mydomain.example.com, example.com", "name": "domain-search", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 900, "dhcp4o6-port": 0, "rebind-timer": 1800, "authoritative": false, "sanity-checks": {"lease-checks": "warn", "extended-info-checks": "fix"}, "boot-file-name": "", "echo-client-id": true, "lease-database": {"type": "memfile", "lfc-interval": 3600}, "valid-lifetime": 3600, "cache-threshold": 0.25, "control-sockets": [{"socket-name": "/var/run/kea/kea4-ctrl-socket", "socket-type": "unix"}], "hooks-libraries": [{"library": "libdhcp_lease_cmds.so"}, {"library": "libdhcp_host_cmds.so"}, {"library": "libdhcp_subnet_cmds.so"}, {"library": "libdhcp_mysql.so"}, {"library": "libdhcp_ha.so", "parameters": {"high-availability": [{"mode": "hot-standby", "peers": [{"url": "http://172.24.0.101:8003", "name": "server1", "role": "primary", "auto-failover": true}, {"url": "http://172.24.0.110:8004", "name": "server2", "role": "standby", "auto-failover": true}], "sync-leases": true, "sync-timeout": 60000, "max-ack-delay": 5000, "heartbeat-delay": 10000, "multi-threading": {"http-client-threads": 4, "http-listener-threads": 4, "enable-multi-threading": true, "http-dedicated-listener": true}, "sync-page-limit": 10000, "wait-backup-ack": false, "this-server-name": "server1", "restrict-commands": true, "max-response-delay": 20000, "send-lease-updates": true, "max-unacked-clients": 3, "require-client-certs": true, "delayed-updates-limit": 0, "max-rejected-lease-updates": 10}]}}], "hosts-databases": [{"host": "mariadb", "name": "agent_kea_ha1", "type": "mysql", "user": "agent_kea_ha1", "password": "agent_kea_ha1"}], "match-client-id": true, "multi-threading": {"thread-pool-size": 0, "packet-queue-size": 64, "enable-multi-threading": true}, "server-hostname": "", "shared-networks": [{"name": "esperanto", "relay": {"ip-addresses": []}, "subnet4": [{"id": 123, "pools": [], "relay": {"ip-addresses": ["172.103.0.200"]}, "subnet": "192.110.111.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 900, "rebind-timer": 1800, "reservations": [{"hostname": "", "hw-address": "01:01:01:01:01:01", "ip-address": "192.110.111.230", "next-server": "0.0.0.0", "option-data": [{"code": 20, "data": "false", "name": "non-local-source-routing", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 222, "data": "", "space": "dhcp4", "csv-format": false, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}], "user-context": {"site": "esperanto", "subnet-name": "valletta"}, "4o6-interface": "", "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}, {"id": 124, "pools": [], "relay": {"ip-addresses": ["172.103.0.200"]}, "subnet": "192.110.112.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 900, "rebind-timer": 1800, "reservations": [], "user-context": {"site": "esperanto", "subnet-name": "vilnius"}, "4o6-interface": "", "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}], "allocator": "iterative", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 900, "rebind-timer": 1800, "valid-lifetime": 3600, "cache-threshold": 0.25, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}], "ddns-send-updates": true, "hostname-char-set": "[^A-Za-z0-9.-]", "interfaces-config": {"re-detect": true, "interfaces": []}, "dhcp-queue-control": {"capacity": 64, "queue-type": "kea-ring4", "enable-queue": false}, "calculate-tee-times": false, "parked-packet-limit": 256, "reservations-global": false, "stash-agent-options": false, "store-extended-info": false, "ddns-update-on-renew": false, "ddns-generated-prefix": "myhost", "ddns-qualifying-suffix": "", "ip-reservations-unique": true, "reservations-in-subnet": true, "ddns-override-no-update": false, "ddns-replace-client-name": "never", "decline-probation-period": 86400, "reservations-out-of-pool": false, "expired-leases-processing": {"max-reclaim-time": 250, "max-reclaim-leases": 100, "hold-reclaimed-time": 3600, "reclaim-timer-wait-time": 10, "unwarned-reclaim-cycles": 5, "flush-reclaimed-timer-wait-time": 25}, "hostname-char-replacement": "", "reservations-lookup-first": false, "ddns-override-client-update": false, "host-reservation-identifiers": ["hw-address", "duid", "circuit-id", "client-id"], "statistic-default-sample-age": 0, "ddns-conflict-resolution-mode": "check-with-dhcid", "statistic-default-sample-count": 20, "early-global-reservations-lookup": false}}', 'e39799467b2a246d9aa6ea243df305d7');
INSERT INTO public.kea_daemon VALUES (13, 13, '{"hash": "1C9ABF367B08D9A323C7F8341DBAD91AC0657E2ACD5B88025FA7CC642A547707", "Dhcp4": {"loggers": [{"name": "kea-dhcp4", "severity": "INFO", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-dhcp4-ha2.log", "maxsize": 10240000, "pattern": ""}]}], "subnet4": [{"id": 1, "pools": [{"pool": "192.0.20.1-192.0.20.200", "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "192.0.20.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [{"code": 3, "data": "192.0.20.1", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 900, "rebind-timer": 1800, "reservations": [{"hostname": "", "hw-address": "00:0c:01:02:03:04", "ip-address": "192.0.20.50", "next-server": "0.0.0.0", "option-data": [{"code": 67, "data": "/tmp/ha-server2/boot.file", "name": "boot-file-name", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 69, "data": "119.12.13.14", "name": "smtp-server", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "00:0c:01:02:03:05", "ip-address": "192.0.20.100", "next-server": "0.0.0.0", "option-data": [{"code": 3, "data": "192.0.20.1", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "00:0c:01:02:03:06", "ip-address": "192.0.20.150", "next-server": "0.0.0.0", "option-data": [{"code": 3, "data": "192.0.20.2", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}], "user-context": {"ha-server-name": "server4"}, "4o6-interface": "", "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}, {"id": 2, "pools": [{"pool": "192.0.3.1-192.0.3.200", "option-data": []}], "relay": {"ip-addresses": []}, "subnet": "192.0.3.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [{"code": 3, "data": "192.0.3.1", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 900, "rebind-timer": 1800, "reservations": [{"hostname": "", "hw-address": "00:0c:01:02:03:04", "ip-address": "192.0.3.50", "next-server": "0.0.0.0", "option-data": [{"code": 67, "data": "/tmp/ha-server2/boot.file", "name": "boot-file-name", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 69, "data": "119.12.13.14", "name": "smtp-server", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "00:0c:01:02:03:05", "ip-address": "192.0.3.100", "next-server": "0.0.0.0", "option-data": [{"code": 3, "data": "192.0.3.1", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}, {"hostname": "", "hw-address": "00:0c:01:02:03:06", "ip-address": "192.0.3.150", "next-server": "0.0.0.0", "option-data": [{"code": 3, "data": "192.0.3.2", "name": "routers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "boot-file-name": "", "client-classes": [], "server-hostname": ""}], "4o6-interface": "", "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}], "allocator": "iterative", "dhcp-ddns": {"sender-ip": "0.0.0.0", "server-ip": "127.0.0.1", "ncr-format": "JSON", "sender-port": 0, "server-port": 53001, "ncr-protocol": "UDP", "enable-updates": false, "max-queue-size": 1024}, "option-def": [], "server-tag": "", "t1-percent": 0.5, "t2-percent": 0.875, "next-server": "0.0.0.0", "option-data": [{"code": 6, "data": "192.0.3.1, 192.0.3.2", "name": "domain-name-servers", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 15, "data": "example.org", "name": "domain-name", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}, {"code": 119, "data": "mydomain.example.com, example.com", "name": "domain-search", "space": "dhcp4", "csv-format": true, "never-send": false, "always-send": false}], "renew-timer": 900, "dhcp4o6-port": 0, "rebind-timer": 1800, "authoritative": false, "sanity-checks": {"lease-checks": "warn", "extended-info-checks": "fix"}, "boot-file-name": "", "echo-client-id": true, "lease-database": {"type": "memfile", "lfc-interval": 3600}, "valid-lifetime": 3600, "cache-threshold": 0.25, "control-sockets": [{"socket-name": "/var/run/kea/kea4-ctrl-socket", "socket-type": "unix"}], "hooks-libraries": [{"library": "libdhcp_lease_cmds.so"}, {"library": "libdhcp_host_cmds.so"}, {"library": "libdhcp_subnet_cmds.so"}, {"library": "libdhcp_mysql.so"}, {"library": "libdhcp_ha.so", "parameters": {"high-availability": [{"mode": "hot-standby", "peers": [{"url": "http://172.24.0.101:8003", "name": "server1", "role": "primary", "auto-failover": true}, {"url": "http://172.24.0.110:8004", "name": "server2", "role": "standby", "auto-failover": true}], "sync-leases": true, "sync-timeout": 60000, "max-ack-delay": 5000, "heartbeat-delay": 10000, "multi-threading": {"http-client-threads": 4, "http-listener-threads": 4, "enable-multi-threading": true, "http-dedicated-listener": true}, "sync-page-limit": 10000, "wait-backup-ack": false, "this-server-name": "server2", "restrict-commands": true, "max-response-delay": 20000, "send-lease-updates": true, "max-unacked-clients": 3, "require-client-certs": true, "delayed-updates-limit": 0, "max-rejected-lease-updates": 10}, {"mode": "hot-standby", "peers": [{"url": "http://172.24.0.121:8005", "name": "server3", "role": "primary", "auto-failover": true}, {"url": "http://172.24.0.110:8006", "name": "server4", "role": "standby", "auto-failover": true}], "sync-leases": true, "sync-timeout": 60000, "max-ack-delay": 5000, "heartbeat-delay": 10000, "multi-threading": {"http-client-threads": 4, "http-listener-threads": 4, "enable-multi-threading": true, "http-dedicated-listener": true}, "sync-page-limit": 10000, "wait-backup-ack": false, "this-server-name": "server4", "restrict-commands": true, "max-response-delay": 20000, "send-lease-updates": true, "max-unacked-clients": 0, "require-client-certs": true, "delayed-updates-limit": 0, "max-rejected-lease-updates": 10}]}}], "hosts-databases": [{"host": "mariadb", "name": "agent_kea_ha2", "type": "mysql", "user": "agent_kea_ha2", "password": "agent_kea_ha2"}], "match-client-id": true, "multi-threading": {"thread-pool-size": 0, "packet-queue-size": 64, "enable-multi-threading": true}, "server-hostname": "", "shared-networks": [{"name": "esperanto", "relay": {"ip-addresses": []}, "subnet4": [{"id": 123, "pools": [], "relay": {"ip-addresses": ["172.103.0.200"]}, "subnet": "192.110.111.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 900, "rebind-timer": 1800, "reservations": [], "user-context": {"site": "esperanto", "subnet-name": "valletta"}, "4o6-interface": "", "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}, {"id": 124, "pools": [], "relay": {"ip-addresses": ["172.103.0.200"]}, "subnet": "192.110.112.0/24", "allocator": "iterative", "4o6-subnet": "", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 900, "rebind-timer": 1800, "reservations": [], "user-context": {"site": "esperanto", "subnet-name": "vilnius"}, "4o6-interface": "", "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}], "allocator": "iterative", "t1-percent": 0.5, "t2-percent": 0.875, "option-data": [], "renew-timer": 900, "rebind-timer": 1800, "valid-lifetime": 3600, "cache-threshold": 0.25, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false, "store-extended-info": false}], "ddns-send-updates": true, "hostname-char-set": "[^A-Za-z0-9.-]", "interfaces-config": {"re-detect": true, "interfaces": []}, "dhcp-queue-control": {"capacity": 64, "queue-type": "kea-ring4", "enable-queue": false}, "calculate-tee-times": false, "parked-packet-limit": 256, "reservations-global": false, "stash-agent-options": false, "store-extended-info": false, "ddns-update-on-renew": false, "ddns-generated-prefix": "myhost", "ddns-qualifying-suffix": "", "ip-reservations-unique": true, "reservations-in-subnet": true, "ddns-override-no-update": false, "ddns-replace-client-name": "never", "decline-probation-period": 86400, "reservations-out-of-pool": false, "expired-leases-processing": {"max-reclaim-time": 250, "max-reclaim-leases": 100, "hold-reclaimed-time": 3600, "reclaim-timer-wait-time": 10, "unwarned-reclaim-cycles": 5, "flush-reclaimed-timer-wait-time": 25}, "hostname-char-replacement": "", "reservations-lookup-first": false, "ddns-override-client-update": false, "host-reservation-identifiers": ["hw-address", "duid", "circuit-id", "client-id"], "statistic-default-sample-age": 0, "ddns-conflict-resolution-mode": "check-with-dhcid", "statistic-default-sample-count": 20, "early-global-reservations-lookup": false}}', 'd570a4cccf0d2d18b7f219ffc3c0f26e');
INSERT INTO public.kea_daemon VALUES (14, 14, NULL, NULL);
INSERT INTO public.kea_daemon VALUES (4, 4, NULL, NULL);
INSERT INTO public.kea_daemon VALUES (1, 1, '{"hash": "EC311D0CAB509F97319039E86BB40CC163E69AEBAB9DD3ABB6E702A4D9E09FC6", "Control-agent": {"loggers": [{"name": "kea-ctrl-agent", "severity": "INFO", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-ctrl-agent.log", "maxsize": 10240000, "pattern": ""}]}], "http-host": "0.0.0.0", "http-port": 8000, "control-sockets": {"d2": {"socket-name": "/var/run/kea/kea-ddns-ctrl-socket", "socket-type": "unix"}, "dhcp4": {"socket-name": "/var/run/kea/kea4-ctrl-socket", "socket-type": "unix"}, "dhcp6": {"socket-name": "/var/run/kea/kea6-ctrl-socket", "socket-type": "unix"}}, "hooks-libraries": []}}', 'e3c0c3ba2d8ae9452ed6d8de5ac7b085');
INSERT INTO public.kea_daemon VALUES (7, 7, '{"hash": "EC311D0CAB509F97319039E86BB40CC163E69AEBAB9DD3ABB6E702A4D9E09FC6", "Control-agent": {"loggers": [{"name": "kea-ctrl-agent", "severity": "INFO", "debuglevel": 0, "output-options": [{"flush": true, "output": "stdout", "pattern": "%-5p %m\n"}, {"flush": true, "maxver": 1, "output": "/var/log/kea/kea-ctrl-agent.log", "maxsize": 10240000, "pattern": ""}]}], "http-host": "0.0.0.0", "http-port": 8000, "control-sockets": {"d2": {"socket-name": "/var/run/kea/kea-ddns-ctrl-socket", "socket-type": "unix"}, "dhcp4": {"socket-name": "/var/run/kea/kea4-ctrl-socket", "socket-type": "unix"}, "dhcp6": {"socket-name": "/var/run/kea/kea6-ctrl-socket", "socket-type": "unix"}}, "hooks-libraries": []}}', 'e3c0c3ba2d8ae9452ed6d8de5ac7b085');


--
-- TOC entry 3912 (class 0 OID 16847)
-- Dependencies: 259
-- Data for Name: kea_dhcp_daemon; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.kea_dhcp_daemon VALUES (4, 9, '{"RPS1": 0, "RPS2": 0}');
INSERT INTO public.kea_dhcp_daemon VALUES (5, 10, '{"RPS1": 0, "RPS2": 0}');
INSERT INTO public.kea_dhcp_daemon VALUES (3, 5, '{"RPS1": 0, "RPS2": 0}');
INSERT INTO public.kea_dhcp_daemon VALUES (8, 15, '{"RPS1": 0, "RPS2": 0}');
INSERT INTO public.kea_dhcp_daemon VALUES (9, 16, '{"RPS1": 0, "RPS2": 0}');
INSERT INTO public.kea_dhcp_daemon VALUES (6, 13, '{"RPS1": 0, "RPS2": 0}');
INSERT INTO public.kea_dhcp_daemon VALUES (7, 14, '{"RPS1": 0, "RPS2": 0}');
INSERT INTO public.kea_dhcp_daemon VALUES (2, 4, '{"RPS1": 0, "RPS2": 0}');
INSERT INTO public.kea_dhcp_daemon VALUES (1, 3, '{"RPS1": 0, "RPS2": 0}');


--
-- TOC entry 3905 (class 0 OID 16773)
-- Dependencies: 252
-- Data for Name: local_host; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.local_host VALUES (1, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 1, NULL);
INSERT INTO public.local_host VALUES (2, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 2, NULL);
INSERT INTO public.local_host VALUES (3, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 3, NULL);
INSERT INTO public.local_host VALUES (4, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 4, NULL);
INSERT INTO public.local_host VALUES (5, '2025-10-31 09:53:44.651209', 'config', 3, '[{"Code": 6, "Name": "domain-name-servers", "Space": "dhcp4", "Fields": [{"Values": ["10.1.1.202"], "FieldType": "ipv4-address"}, {"Values": ["10.1.1.203"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '92f43733000b5c24d6cb61d6c79d004a', '{}', '0.0.0.0', NULL, NULL, 5, NULL);
INSERT INTO public.local_host VALUES (6, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '192.0.2.1', 'hal9000', '/dev/null', 6, NULL);
INSERT INTO public.local_host VALUES (7, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 7, 'special-snowflake');
INSERT INTO public.local_host VALUES (8, '2025-10-31 09:53:44.651209', 'config', 3, '[{"Code": 125, "Name": "vivso-suboptions", "Space": "dhcp4", "Fields": [{"Values": [4491], "FieldType": "uint32"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}, {"Code": 2, "Name": "tftp-servers", "Space": "vendor-4491", "Fields": [{"Values": ["10.1.1.202"], "FieldType": "ipv4-address"}, {"Values": ["10.1.1.203"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": "vendor-4491.2"}]', 'fafa310857b73c87067d8a783e85afe8', '{}', '0.0.0.0', NULL, NULL, 8, NULL);
INSERT INTO public.local_host VALUES (9, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 9, NULL);
INSERT INTO public.local_host VALUES (10, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 10, NULL);
INSERT INTO public.local_host VALUES (11, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 11, NULL);
INSERT INTO public.local_host VALUES (12, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 12, NULL);
INSERT INTO public.local_host VALUES (13, '2025-10-31 09:53:44.651209', 'config', 3, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 13, NULL);
INSERT INTO public.local_host VALUES (14, '2025-10-31 09:53:45.426795', 'config', 5, '[{"Code": 23, "Name": "dns-servers", "Space": "dhcp6", "Fields": [{"Values": ["3000:1::234"], "FieldType": "ipv6-address"}], "Universe": 6, "AlwaysSend": false, "Encapsulate": ""}, {"Code": 27, "Name": "nis-servers", "Space": "dhcp6", "Fields": [{"Values": ["3000:1::234"], "FieldType": "ipv6-address"}], "Universe": 6, "AlwaysSend": false, "Encapsulate": ""}]', '71181303f526d9b6f123e7a599c9a239', '{special_snowflake,office}', NULL, NULL, NULL, 14, NULL);
INSERT INTO public.local_host VALUES (15, '2025-10-31 09:53:45.426795', 'config', 5, '[]', NULL, '{}', NULL, NULL, NULL, 15, NULL);
INSERT INTO public.local_host VALUES (16, '2025-10-31 09:53:45.426795', 'config', 5, '[{"Code": 17, "Name": "vendor-opts", "Space": "dhcp6", "Fields": [{"Values": [4491], "FieldType": "uint32"}], "Universe": 6, "AlwaysSend": false, "Encapsulate": ""}, {"Code": 32, "Name": "tftp-servers", "Space": "vendor-4491", "Fields": [{"Values": ["3000:1::234"], "FieldType": "ipv6-address"}], "Universe": 6, "AlwaysSend": false, "Encapsulate": "vendor-4491.32"}]', 'ff36681bcac9f3c7e047e392df0cad1e', '{}', NULL, NULL, NULL, 16, 'foo.example.com');
INSERT INTO public.local_host VALUES (17, '2025-10-31 09:53:45.426795', 'config', 5, '[]', NULL, '{}', NULL, NULL, NULL, 17, NULL);
INSERT INTO public.local_host VALUES (18, '2025-10-31 09:53:45.426795', 'config', 5, '[]', NULL, '{}', NULL, NULL, NULL, 18, NULL);
INSERT INTO public.local_host VALUES (19, '2025-10-31 09:53:46.547234', 'config', 9, '[{"Code": 67, "Name": "boot-file-name", "Space": "dhcp4", "Fields": [{"Values": ["/tmp/ha-server1/boot.file"], "FieldType": "string"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}, {"Code": 69, "Name": "smtp-server", "Space": "dhcp4", "Fields": [{"Values": ["119.12.13.14"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '891e17ca3a1d3250c2c4dd0df01576a7', '{}', '0.0.0.0', NULL, NULL, 19, NULL);
INSERT INTO public.local_host VALUES (19, '2025-10-31 09:53:46.547234', 'config', 13, '[{"Code": 67, "Name": "boot-file-name", "Space": "dhcp4", "Fields": [{"Values": ["/tmp/ha-server2/boot.file"], "FieldType": "string"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}, {"Code": 69, "Name": "smtp-server", "Space": "dhcp4", "Fields": [{"Values": ["119.12.13.14"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '5f66dd7363aa7b8d1ae017647912e3d6', '{}', '0.0.0.0', NULL, NULL, 22, NULL);
INSERT INTO public.local_host VALUES (20, '2025-10-31 09:53:46.547234', 'config', 9, '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.20.1"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', 'b38284d7b31ce334763114d91073bd23', '{}', '0.0.0.0', NULL, NULL, 20, NULL);
INSERT INTO public.local_host VALUES (20, '2025-10-31 09:53:46.547234', 'config', 13, '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.20.1"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', 'b38284d7b31ce334763114d91073bd23', '{}', '0.0.0.0', NULL, NULL, 23, NULL);
INSERT INTO public.local_host VALUES (21, '2025-10-31 09:53:46.547234', 'config', 9, '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.20.2"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', 'cab575e30801a5eaa807033d371129a0', '{}', '0.0.0.0', NULL, NULL, 21, NULL);
INSERT INTO public.local_host VALUES (21, '2025-10-31 09:53:46.547234', 'config', 13, '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.20.2"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', 'cab575e30801a5eaa807033d371129a0', '{}', '0.0.0.0', NULL, NULL, 24, NULL);
INSERT INTO public.local_host VALUES (22, '2025-10-31 09:53:47.037418', 'config', 13, '[{"Code": 67, "Name": "boot-file-name", "Space": "dhcp4", "Fields": [{"Values": ["/tmp/ha-server2/boot.file"], "FieldType": "string"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}, {"Code": 69, "Name": "smtp-server", "Space": "dhcp4", "Fields": [{"Values": ["119.12.13.14"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '5f66dd7363aa7b8d1ae017647912e3d6', '{}', '0.0.0.0', NULL, NULL, 25, NULL);
INSERT INTO public.local_host VALUES (22, '2025-10-31 09:53:47.037418', 'config', 15, '[{"Code": 67, "Name": "boot-file-name", "Space": "dhcp4", "Fields": [{"Values": ["/tmp/ha-server1/boot.file"], "FieldType": "string"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}, {"Code": 69, "Name": "smtp-server", "Space": "dhcp4", "Fields": [{"Values": ["119.12.13.14"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '891e17ca3a1d3250c2c4dd0df01576a7', '{}', '0.0.0.0', NULL, NULL, 29, NULL);
INSERT INTO public.local_host VALUES (23, '2025-10-31 09:53:47.037418', 'config', 13, '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.3.1"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '9dbd180b71c29cddf1e22773a163a1b2', '{}', '0.0.0.0', NULL, NULL, 26, NULL);
INSERT INTO public.local_host VALUES (23, '2025-10-31 09:53:47.037418', 'config', 15, '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.3.1"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '9dbd180b71c29cddf1e22773a163a1b2', '{}', '0.0.0.0', NULL, NULL, 30, NULL);
INSERT INTO public.local_host VALUES (24, '2025-10-31 09:53:47.037418', 'config', 13, '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.3.2"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '7aa6b879209fd749c4a0c3427c22f7f1', '{}', '0.0.0.0', NULL, NULL, 27, NULL);
INSERT INTO public.local_host VALUES (24, '2025-10-31 09:53:47.037418', 'config', 15, '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.3.2"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '7aa6b879209fd749c4a0c3427c22f7f1', '{}', '0.0.0.0', NULL, NULL, 31, NULL);
INSERT INTO public.local_host VALUES (26, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 35, NULL);
INSERT INTO public.local_host VALUES (26, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 51, NULL);
INSERT INTO public.local_host VALUES (26, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 67, NULL);
INSERT INTO public.local_host VALUES (27, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 36, NULL);
INSERT INTO public.local_host VALUES (27, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 52, NULL);
INSERT INTO public.local_host VALUES (27, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 68, NULL);
INSERT INTO public.local_host VALUES (28, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 37, NULL);
INSERT INTO public.local_host VALUES (28, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 53, NULL);
INSERT INTO public.local_host VALUES (28, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 69, NULL);
INSERT INTO public.local_host VALUES (25, '2025-10-31 09:53:56.606348', 'config', 15, '[{"Code": 20, "Name": "non-local-source-routing", "Space": "dhcp4", "Fields": [{"Values": [false], "FieldType": "bool"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}, {"Code": 222, "Name": "", "Space": "dhcp4", "Fields": null, "Universe": 4, "AlwaysSend": false, "Encapsulate": "option-222"}]', '94760ce5816f8e0ca4acb1dd5c491ec6', '{}', '0.0.0.0', NULL, NULL, 28, NULL);
INSERT INTO public.local_host VALUES (25, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 38, NULL);
INSERT INTO public.local_host VALUES (25, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 54, NULL);
INSERT INTO public.local_host VALUES (25, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 70, NULL);
INSERT INTO public.local_host VALUES (29, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 39, 'fish.example.org');
INSERT INTO public.local_host VALUES (29, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 55, 'fish.example.org');
INSERT INTO public.local_host VALUES (29, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 71, 'fish.example.org');
INSERT INTO public.local_host VALUES (30, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 40, 'gibberish');
INSERT INTO public.local_host VALUES (30, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 56, 'gibberish');
INSERT INTO public.local_host VALUES (30, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 72, 'gibberish');
INSERT INTO public.local_host VALUES (31, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 41, NULL);
INSERT INTO public.local_host VALUES (31, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 57, NULL);
INSERT INTO public.local_host VALUES (31, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 73, NULL);
INSERT INTO public.local_host VALUES (32, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 42, NULL);
INSERT INTO public.local_host VALUES (32, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 58, NULL);
INSERT INTO public.local_host VALUES (32, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 74, NULL);
INSERT INTO public.local_host VALUES (33, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 43, NULL);
INSERT INTO public.local_host VALUES (33, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 59, NULL);
INSERT INTO public.local_host VALUES (33, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 75, NULL);
INSERT INTO public.local_host VALUES (34, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 44, NULL);
INSERT INTO public.local_host VALUES (34, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 60, NULL);
INSERT INTO public.local_host VALUES (34, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 76, NULL);
INSERT INTO public.local_host VALUES (35, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 45, NULL);
INSERT INTO public.local_host VALUES (35, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 61, NULL);
INSERT INTO public.local_host VALUES (35, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 77, NULL);
INSERT INTO public.local_host VALUES (36, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 46, NULL);
INSERT INTO public.local_host VALUES (36, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 62, NULL);
INSERT INTO public.local_host VALUES (36, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 78, NULL);
INSERT INTO public.local_host VALUES (37, '2025-10-31 09:53:56.606348', 'api', 9, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 47, NULL);
INSERT INTO public.local_host VALUES (37, '2025-10-31 09:53:56.606348', 'api', 13, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 63, NULL);
INSERT INTO public.local_host VALUES (37, '2025-10-31 09:53:56.606348', 'api', 15, '[]', NULL, '{}', '0.0.0.0', NULL, NULL, 79, NULL);


--
-- TOC entry 3931 (class 0 OID 17069)
-- Dependencies: 278
-- Data for Name: local_shared_network; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.local_shared_network VALUES (3, 1, '{"Relay": {"ip-addresses": ["172.101.0.200"]}, "Allocator": "iterative", "Interface": null, "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 200, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL);
INSERT INTO public.local_shared_network VALUES (3, 2, '{"Relay": {"ip-addresses": ["172.102.0.200"]}, "Allocator": "iterative", "Interface": null, "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 200, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL);
INSERT INTO public.local_shared_network VALUES (5, 3, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": null, "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": false, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 300, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL);
INSERT INTO public.local_shared_network VALUES (9, 4, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": null, "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 3600, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[]', NULL);
INSERT INTO public.local_shared_network VALUES (13, 4, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": null, "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 3600, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[]', NULL);
INSERT INTO public.local_shared_network VALUES (15, 4, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": null, "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 3600, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[]', NULL);


--
-- TOC entry 3896 (class 0 OID 16644)
-- Dependencies: 243
-- Data for Name: local_subnet; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.local_subnet VALUES (8, 21, '2025-10-31 09:59:57.012307', '{"total-addresses": "50", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.102.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-02-00"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 8, NULL);
INSERT INTO public.local_subnet VALUES (4, 14, '2025-10-31 09:59:57.014029', '{"total-addresses": "50", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.101.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-01-03"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 4, '{"subnet-name": "bob"}');
INSERT INTO public.local_subnet VALUES (9, 22, '2025-10-31 09:59:57.010993', '{"total-addresses": "150", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.102.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-02-01"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 9, NULL);
INSERT INTO public.local_subnet VALUES (10, 23, '2025-10-31 09:59:57.012007', '{"total-addresses": "245", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.102.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-02-02"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 10, NULL);
INSERT INTO public.local_subnet VALUES (2, 12, '2025-10-31 09:59:57.014387', '{"total-addresses": "110", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.101.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-01-01"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 2, '{}');
INSERT INTO public.local_subnet VALUES (20, 2, '2025-10-31 09:59:57.032067', '{"total-nas": "281474976710656", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": false, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-30-00"], "valid-lifetime": 4000, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 4000, "min-valid-lifetime": 4000, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL, 20, NULL);
INSERT INTO public.local_subnet VALUES (21, 10, '2025-10-31 09:59:57.032479', '{"total-nas": "4", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": false, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-30-01"], "valid-lifetime": 4000, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 4000, "min-valid-lifetime": 4000, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL, 21, NULL);
INSERT INTO public.local_subnet VALUES (24, 1, '2025-10-31 09:59:57.043898', '{"total-addresses": "200", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 9, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.20.1"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', 'b38284d7b31ce334763114d91073bd23', 24, '{"ha-server-name": "server3"}');
INSERT INTO public.local_subnet VALUES (22, 123, '2025-10-31 09:59:57.043371', '{"total-addresses": "0", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 9, '{"Relay": {"ip-addresses": ["172.103.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[]', NULL, 22, '{"site": "esperanto", "subnet-name": "valletta"}');
INSERT INTO public.local_subnet VALUES (22, 123, '2025-10-31 09:59:57.049788', '{"total-addresses": "0", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 13, '{"Relay": {"ip-addresses": ["172.103.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[]', NULL, 25, '{"site": "esperanto", "subnet-name": "valletta"}');
INSERT INTO public.local_subnet VALUES (24, 1, '2025-10-31 09:59:57.050035', '{"total-addresses": "200", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 13, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.20.1"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', 'b38284d7b31ce334763114d91073bd23', 27, '{"ha-server-name": "server4"}');
INSERT INTO public.local_subnet VALUES (11, 1, '2025-10-31 09:59:57.01168', '{"total-addresses": "200", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "2", "declined-addresses": "1", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.100.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-00-00"], "valid-lifetime": 180, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 180, "min-valid-lifetime": 180, "calculate-tee-times": false}', '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.2.1"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '32d88bd8a4c01391c650c428256b82df', 11, NULL);
INSERT INTO public.local_subnet VALUES (5, 15, '2025-10-31 09:59:57.012974', '{"total-addresses": "50", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.101.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-01-04"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 5, NULL);
INSERT INTO public.local_subnet VALUES (16, 7, '2025-10-31 09:59:57.034113', '{"total-nas": "36893488147419103232", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": null, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-50-03"], "valid-lifetime": 300, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL, 16, NULL);
INSERT INTO public.local_subnet VALUES (7, 17, '2025-10-31 09:59:57.013285', '{"total-addresses": "3", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.101.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-01-06"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 7, NULL);
INSERT INTO public.local_subnet VALUES (6, 16, '2025-10-31 09:59:57.014666', '{"total-addresses": "50", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.101.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-01-05"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 6, NULL);
INSERT INTO public.local_subnet VALUES (23, 124, '2025-10-31 09:59:57.056192', '{"total-addresses": "0", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 15, '{"Relay": {"ip-addresses": ["172.103.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[]', NULL, 30, '{"site": "esperanto", "subnet-name": "vilnius"}');
INSERT INTO public.local_subnet VALUES (25, 2, '2025-10-31 09:59:57.049177', '{"total-addresses": "200", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 13, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.3.1"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '9dbd180b71c29cddf1e22773a163a1b2', 28, NULL);
INSERT INTO public.local_subnet VALUES (23, 124, '2025-10-31 09:59:57.043684', '{"total-addresses": "0", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 9, '{"Relay": {"ip-addresses": ["172.103.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[]', NULL, 23, '{"site": "esperanto", "subnet-name": "vilnius"}');
INSERT INTO public.local_subnet VALUES (23, 124, '2025-10-31 09:59:57.049532', '{"total-addresses": "0", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 13, '{"Relay": {"ip-addresses": ["172.103.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[]', NULL, 26, '{"site": "esperanto", "subnet-name": "vilnius"}');
INSERT INTO public.local_subnet VALUES (25, 1, '2025-10-31 09:59:57.056383', '{"total-addresses": "200", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 15, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[{"Code": 3, "Name": "routers", "Space": "dhcp4", "Fields": [{"Values": ["192.0.3.1"], "FieldType": "ipv4-address"}], "Universe": 4, "AlwaysSend": false, "Encapsulate": ""}]', '9dbd180b71c29cddf1e22773a163a1b2', 31, NULL);
INSERT INTO public.local_subnet VALUES (17, 8, '2025-10-31 09:59:57.033383', '{"total-nas": "36893488147419103232", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": null, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-50-04"], "valid-lifetime": 300, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL, 17, NULL);
INSERT INTO public.local_subnet VALUES (12, 3, '2025-10-31 09:59:57.032696', '{"total-nas": "281474976710656", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": null, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-40-01"], "valid-lifetime": 300, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL, 12, NULL);
INSERT INTO public.local_subnet VALUES (15, 6, '2025-10-31 09:59:57.03385', '{"total-nas": "36893488147419103232", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": null, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-50-02"], "valid-lifetime": 300, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL, 15, NULL);
INSERT INTO public.local_subnet VALUES (13, 4, '2025-10-31 09:59:57.032939', '{"total-nas": "36893488147419103232", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": null, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-50-00"], "valid-lifetime": 300, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL, 13, NULL);
INSERT INTO public.local_subnet VALUES (19, 1, '2025-10-31 09:59:57.033175', '{"total-nas": "844424930131968", "total-pds": "512", "assigned-nas": "2", "assigned-pds": "1", "declined-nas": "1", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": false, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-30-01"], "valid-lifetime": 4000, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 4000, "min-valid-lifetime": 4000, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[{"Code": 23, "Name": "dns-servers", "Space": "dhcp6", "Fields": [{"Values": ["3001:db8:2::dead:beef"], "FieldType": "ipv6-address"}, {"Values": ["3001:db8:2::cafe:babe"], "FieldType": "ipv6-address"}], "Universe": 6, "AlwaysSend": false, "Encapsulate": ""}]', 'c28cec5caab3e913eb1448d5bb3fe9b1', 19, NULL);
INSERT INTO public.local_subnet VALUES (1, 11, '2025-10-31 09:59:57.013577', '{"total-addresses": "50", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.101.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-01-00"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 1, '{"baz": 42, "boz": ["a", "b", "c"], "foo": "bar"}');
INSERT INTO public.local_subnet VALUES (22, 123, '2025-10-31 09:59:57.055899', '{"total-addresses": "0", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 15, '{"Relay": {"ip-addresses": ["172.103.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 900, "BootFileName": null, "rebind-timer": 1800, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-03-00"], "valid-lifetime": 3600, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 3600, "min-valid-lifetime": 3600, "calculate-tee-times": false}', '[]', NULL, 29, '{"site": "esperanto", "subnet-name": "valletta"}');
INSERT INTO public.local_subnet VALUES (14, 5, '2025-10-31 09:59:57.033595', '{"total-nas": "36893488147419103232", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": null, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-50-01"], "valid-lifetime": 300, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL, 14, NULL);
INSERT INTO public.local_subnet VALUES (18, 9, '2025-10-31 09:59:57.034337', '{"total-nas": "36893488147419103232", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "registered-nas": "0", "reclaimed-leases": "0", "v6-ia-na-lease-reuses": "0", "v6-ia-pd-lease-reuses": "0", "cumulative-assigned-nas": "0", "cumulative-assigned-pds": "0", "cumulative-registered-nas": "0", "reclaimed-declined-addresses": "0"}', 5, '{"Relay": {"ip-addresses": []}, "Allocator": "iterative", "Interface": "eth1", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.8, "InterfaceID": null, "PDAllocator": "iterative", "RapidCommit": null, "renew-timer": 1000, "BootFileName": null, "rebind-timer": 2000, "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-50-05"], "valid-lifetime": 300, "cache-threshold": 0.25, "StoreExtendedInfo": false, "max-valid-lifetime": 300, "min-valid-lifetime": 300, "preferred-lifetime": 3000, "calculate-tee-times": true, "max-preferred-lifetime": 3000, "min-preferred-lifetime": 3000}', '[]', NULL, 18, NULL);
INSERT INTO public.local_subnet VALUES (3, 13, '2025-10-31 09:59:57.012673', '{"total-addresses": "100", "v4-lease-reuses": "0", "reclaimed-leases": "0", "assigned-addresses": "0", "declined-addresses": "0", "v4-reservation-conflicts": "0", "reclaimed-declined-addresses": "0", "cumulative-assigned-addresses": "0"}', 3, '{"Relay": {"ip-addresses": ["172.101.0.200"]}, "Allocator": "iterative", "Interface": null, "4o6-subnet": "", "NextServer": null, "t1-percent": 0.5, "t2-percent": 0.875, "InterfaceID": null, "PDAllocator": null, "RapidCommit": null, "renew-timer": 90, "BootFileName": null, "rebind-timer": 120, "4o6-interface": "", "Authoritative": null, "MatchClientID": null, "ServerHostname": null, "client-classes": ["class-01-02"], "valid-lifetime": 200, "cache-threshold": 0.25, "4o6-interface-id": "", "StoreExtendedInfo": false, "max-valid-lifetime": 200, "min-valid-lifetime": 200, "calculate-tee-times": false}', '[]', NULL, 3, '{"subnet-name": "alice"}');


--
-- TOC entry 3938 (class 0 OID 17166)
-- Dependencies: 285
-- Data for Name: local_zone; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.local_zone VALUES (1, 19, 1, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (2, 19, 2, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (3, 19, 3, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (4, 19, 4, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (5, 19, 5, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (6, 19, 6, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (7, 19, 7, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (8, 19, 8, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (9, 19, 9, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (10, 19, 10, 'localhost', 'IN', 2024031501, 'master', '2025-10-31 09:49:03', NULL, false);
INSERT INTO public.local_zone VALUES (11, 20, 11, '_bind', 'CH', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (12, 20, 12, '_bind', 'CH', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (13, 20, 13, '_bind', 'CH', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (14, 20, 14, '_bind', 'CH', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (15, 20, 15, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (16, 20, 16, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (17, 20, 17, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (18, 20, 18, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (19, 20, 19, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (20, 20, 20, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (21, 20, 21, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (22, 20, 22, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (23, 20, 23, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (24, 20, 24, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (25, 20, 25, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (26, 20, 26, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (27, 20, 27, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (28, 20, 28, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (29, 20, 29, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (30, 20, 30, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (31, 20, 31, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (32, 20, 32, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (33, 20, 33, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (34, 20, 34, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (35, 20, 35, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (36, 20, 36, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (37, 20, 37, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (38, 20, 38, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (39, 20, 39, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (40, 20, 40, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (41, 20, 41, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (42, 20, 42, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (43, 20, 43, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (44, 20, 44, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (45, 20, 45, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (46, 20, 46, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (47, 20, 47, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (48, 20, 48, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (49, 20, 49, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (50, 20, 50, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (51, 20, 51, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (52, 20, 52, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (53, 20, 53, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (54, 20, 54, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (55, 20, 55, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (56, 20, 56, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (57, 20, 57, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (58, 20, 58, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (59, 20, 59, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (60, 20, 60, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (61, 20, 61, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (62, 20, 62, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (63, 20, 63, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (64, 20, 64, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (65, 20, 65, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (66, 20, 66, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (67, 20, 67, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (68, 20, 68, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (69, 20, 69, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (70, 20, 70, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (71, 20, 71, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (72, 20, 72, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (73, 20, 73, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (74, 20, 74, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (75, 20, 75, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (76, 20, 76, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (77, 20, 77, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (78, 20, 78, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (79, 20, 79, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (80, 20, 80, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (81, 20, 81, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (82, 20, 82, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (83, 20, 83, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (84, 20, 84, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (85, 20, 85, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (86, 20, 86, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (87, 20, 87, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (88, 20, 88, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (89, 20, 89, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (90, 20, 90, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (91, 20, 91, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (92, 20, 92, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (93, 20, 93, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (94, 20, 94, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (95, 20, 95, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (96, 20, 96, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (97, 20, 97, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (98, 20, 98, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (99, 20, 99, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (100, 20, 100, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (101, 20, 101, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (102, 20, 102, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (103, 20, 103, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (104, 20, 104, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (105, 20, 105, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (106, 20, 106, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (107, 20, 107, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (108, 20, 108, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (109, 20, 109, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (110, 20, 110, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (111, 20, 111, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (112, 20, 112, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (113, 20, 113, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (114, 20, 114, 'guest', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (115, 20, 115, 'guest', 'IN', 2024031501, 'primary', '2025-06-11 14:59:14', NULL, false);
INSERT INTO public.local_zone VALUES (116, 20, 15, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (117, 20, 16, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (118, 20, 17, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (119, 20, 18, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (120, 20, 19, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (121, 20, 20, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (122, 20, 21, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (123, 20, 22, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (124, 20, 23, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (125, 20, 24, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (126, 20, 25, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (127, 20, 26, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (128, 20, 27, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (129, 20, 28, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (130, 20, 29, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (131, 20, 30, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (132, 20, 31, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (133, 20, 32, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (134, 20, 33, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (135, 20, 34, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (136, 20, 35, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (137, 20, 36, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (138, 20, 37, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (139, 20, 38, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (140, 20, 39, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (141, 20, 40, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (142, 20, 41, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (143, 20, 42, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (144, 20, 43, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (145, 20, 44, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (146, 20, 45, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (147, 20, 46, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (148, 20, 47, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (149, 20, 48, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (150, 20, 49, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (151, 20, 50, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (152, 20, 51, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (153, 20, 52, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (154, 20, 53, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (155, 20, 54, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (156, 20, 55, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (157, 20, 56, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (158, 20, 57, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (159, 20, 58, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (160, 20, 59, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (161, 20, 60, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (162, 20, 61, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (163, 20, 62, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (164, 20, 63, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (165, 20, 64, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (166, 20, 65, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (167, 20, 66, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (168, 20, 67, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (169, 20, 68, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (170, 20, 69, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (171, 20, 70, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (172, 20, 71, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (173, 20, 72, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (174, 20, 73, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (175, 20, 74, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (176, 20, 75, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (177, 20, 76, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (178, 20, 77, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (179, 20, 78, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (180, 20, 79, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (181, 20, 80, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (182, 20, 81, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (183, 20, 82, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (184, 20, 83, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (185, 20, 84, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (186, 20, 85, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (187, 20, 86, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (188, 20, 87, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (189, 20, 88, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (190, 20, 89, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (191, 20, 90, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (192, 20, 91, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (193, 20, 92, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (194, 20, 93, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (195, 20, 94, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (196, 20, 95, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (197, 20, 96, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (198, 20, 97, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (199, 20, 98, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (200, 20, 99, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (201, 20, 100, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (202, 20, 101, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (203, 20, 102, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (204, 20, 103, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (205, 20, 104, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (206, 20, 105, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (207, 20, 106, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (208, 20, 107, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (209, 20, 108, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (210, 20, 109, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (211, 20, 110, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (212, 20, 111, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (213, 20, 112, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (214, 20, 113, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (215, 20, 114, 'trusted', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (216, 20, 217, 'trusted', 'IN', 2024031501, 'primary', '2025-06-11 14:59:14', NULL, false);
INSERT INTO public.local_zone VALUES (217, 20, 218, 'trusted', 'IN', 201702121, 'primary', '2025-10-03 10:51:09', NULL, true);
INSERT INTO public.local_zone VALUES (218, 21, 11, '_bind', 'CH', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (219, 21, 12, '_bind', 'CH', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (220, 21, 13, '_bind', 'CH', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (221, 21, 14, '_bind', 'CH', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (222, 21, 222, '_default', 'IN', 0, 'mirror', '1970-01-01 00:00:00', NULL, false);
INSERT INTO public.local_zone VALUES (223, 21, 15, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (224, 21, 16, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (225, 21, 17, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (226, 21, 18, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (227, 21, 19, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (228, 21, 20, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (229, 21, 21, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (230, 21, 22, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (231, 21, 23, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (232, 21, 24, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (233, 21, 25, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (234, 21, 26, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (235, 21, 27, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (236, 21, 28, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (237, 21, 29, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (238, 21, 30, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (239, 21, 31, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (240, 21, 32, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (241, 21, 33, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (242, 21, 34, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (243, 21, 35, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (244, 21, 36, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (245, 21, 37, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (246, 21, 38, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (247, 21, 39, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (248, 21, 40, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (249, 21, 41, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (250, 21, 42, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (251, 21, 43, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (252, 21, 44, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (253, 21, 45, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (254, 21, 46, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (255, 21, 47, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (256, 21, 48, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (257, 21, 49, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (258, 21, 50, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (259, 21, 51, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (260, 21, 52, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (261, 21, 53, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (262, 21, 54, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (263, 21, 55, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (264, 21, 56, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (265, 21, 57, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (266, 21, 58, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (267, 21, 59, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (268, 21, 60, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (269, 21, 61, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (270, 21, 62, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (271, 21, 63, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (272, 21, 64, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (273, 21, 65, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (274, 21, 66, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (275, 21, 67, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (276, 21, 68, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (277, 21, 69, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (278, 21, 70, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (279, 21, 71, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (280, 21, 72, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (281, 21, 73, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (282, 21, 74, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (283, 21, 75, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (284, 21, 76, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (285, 21, 77, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (286, 21, 78, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (287, 21, 79, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (288, 21, 80, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (289, 21, 81, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (290, 21, 82, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (291, 21, 83, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (292, 21, 84, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (293, 21, 85, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (294, 21, 86, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (295, 21, 87, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (296, 21, 88, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (297, 21, 89, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (298, 21, 90, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (299, 21, 91, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (300, 21, 92, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (301, 21, 93, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (302, 21, 94, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (303, 21, 95, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (304, 21, 96, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (305, 21, 97, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (306, 21, 98, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (307, 21, 99, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (308, 21, 100, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (309, 21, 101, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (310, 21, 102, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (311, 21, 103, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (312, 21, 104, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (313, 21, 105, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (314, 21, 106, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (315, 21, 107, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (316, 21, 108, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (317, 21, 109, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (318, 21, 110, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (319, 21, 111, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (320, 21, 112, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (321, 21, 113, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (322, 21, 114, '_default', 'IN', 0, 'builtin', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (323, 21, 218, '_default', 'IN', 201702121, 'secondary', '2025-10-31 09:48:54', NULL, true);
INSERT INTO public.local_zone VALUES (324, 21, 324, '_default', 'IN', 201702121, 'primary', '2025-10-03 10:51:09', NULL, true);
INSERT INTO public.local_zone VALUES (325, 21, 115, '_default', 'IN', 2024031401, 'secondary', '2025-10-31 09:48:54', NULL, false);
INSERT INTO public.local_zone VALUES (326, 21, 326, '_default', 'IN', 1, 'primary', '2025-06-11 14:59:14', NULL, false);


--
-- TOC entry 3944 (class 0 OID 17227)
-- Dependencies: 291
-- Data for Name: local_zone_rr; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- TOC entry 3918 (class 0 OID 16908)
-- Dependencies: 265
-- Data for Name: log_target; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.log_target VALUES (9, 7, '2025-10-31 09:53:45.842424', 'kea-ctrl-agent', 'info', 'stdout');
INSERT INTO public.log_target VALUES (10, 7, '2025-10-31 09:53:45.842424', 'kea-ctrl-agent', 'info', '/var/log/kea/kea-ctrl-agent.log');
INSERT INTO public.log_target VALUES (11, 9, '2025-10-31 09:53:45.842424', 'kea-dhcp4', 'info', 'stdout');
INSERT INTO public.log_target VALUES (12, 9, '2025-10-31 09:53:45.842424', 'kea-dhcp4', 'info', '/var/log/kea/kea-dhcp4-ha1.log');
INSERT INTO public.log_target VALUES (7, 6, '2025-10-31 09:53:45.426795', 'kea-ctrl-agent', 'info', 'stdout');
INSERT INTO public.log_target VALUES (8, 6, '2025-10-31 09:53:45.426795', 'kea-ctrl-agent', 'info', '/var/log/kea/kea-ctrl-agent.log');
INSERT INTO public.log_target VALUES (5, 5, '2025-10-31 09:53:45.426795', 'kea-dhcp6', 'info', 'stdout');
INSERT INTO public.log_target VALUES (6, 5, '2025-10-31 09:53:45.426795', 'kea-dhcp6', 'info', '/var/log/kea/kea-dhcp6.log');
INSERT INTO public.log_target VALUES (19, 17, '2025-10-31 09:53:47.037418', 'kea-ctrl-agent', 'info', 'stdout');
INSERT INTO public.log_target VALUES (20, 17, '2025-10-31 09:53:47.037418', 'kea-ctrl-agent', 'info', '/var/log/kea/kea-ctrl-agent.log');
INSERT INTO public.log_target VALUES (17, 15, '2025-10-31 09:53:47.037418', 'kea-dhcp4', 'info', 'stdout');
INSERT INTO public.log_target VALUES (18, 15, '2025-10-31 09:53:47.037418', 'kea-dhcp4', 'info', '/var/log/kea/kea-dhcp4-ha1.log');
INSERT INTO public.log_target VALUES (13, 11, '2025-10-31 09:53:46.547234', 'kea-ctrl-agent', 'info', 'stdout');
INSERT INTO public.log_target VALUES (14, 11, '2025-10-31 09:53:46.547234', 'kea-ctrl-agent', 'info', '/var/log/kea/kea-ctrl-agent.log');
INSERT INTO public.log_target VALUES (15, 13, '2025-10-31 09:53:46.547234', 'kea-dhcp4', 'info', 'stdout');
INSERT INTO public.log_target VALUES (16, 13, '2025-10-31 09:53:46.547234', 'kea-dhcp4', 'info', '/var/log/kea/kea-dhcp4-ha2.log');
INSERT INTO public.log_target VALUES (1, 1, '2025-10-31 09:53:44.651209', 'kea-ctrl-agent', 'info', 'stdout');
INSERT INTO public.log_target VALUES (2, 1, '2025-10-31 09:53:44.651209', 'kea-ctrl-agent', 'info', '/var/log/kea/kea-ctrl-agent.log');
INSERT INTO public.log_target VALUES (3, 3, '2025-10-31 09:53:44.651209', 'kea-dhcp4', 'debug', 'stdout');
INSERT INTO public.log_target VALUES (4, 3, '2025-10-31 09:53:44.651209', 'kea-dhcp4', 'debug', '/var/log/kea/kea-dhcp4.log');


--
-- TOC entry 3875 (class 0 OID 16452)
-- Dependencies: 222
-- Data for Name: machine; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.machine VALUES (6, '2025-10-31 09:49:03.819785', 'agent-kea-large', 8884, '{"Os": "", "Cpus": 0, "HostID": "", "Memory": 0, "Uptime": 0, "CpusLoad": "", "Hostname": "", "Platform": "", "KernelArch": "", "UsedMemory": 0, "AgentVersion": "", "KernelVersion": "", "PlatformFamily": "", "PlatformVersion": "", "VirtualizationRole": "", "VirtualizationSystem": ""}', NULL, NULL, 'C6E3D8C08AC1BC77E9F3C0DD8EAC382543B9BE96660EF2B02C7AB5F9E4506649', '\x37e771d40b5f0fc9b8e4567a6203458826f1fbca6c705f2c94a9ab1115b02774', false);
INSERT INTO public.machine VALUES (1, '2025-10-31 09:49:00.231938', 'agent-kea6', 8887, '{"Os": "linux", "Cpus": 6, "HostID": "7e916d04-3bed-ca44-b334-3e2f68ba44ff", "Memory": 7, "Uptime": 4, "CpusLoad": "0.12 0.33 0.53", "Hostname": "agent-kea6", "Platform": "debian", "KernelArch": "aarch64", "UsedMemory": 27, "AgentVersion": "2.3.0", "KernelVersion": "6.8.0-50-generic", "PlatformFamily": "debian", "PlatformVersion": "12.11", "VirtualizationRole": "guest", "VirtualizationSystem": "docker"}', '2025-10-31 10:00:37.779736', NULL, '94A10FBE6CDB7A025BC5FEBEC8B9A637992D02062548E98BF72917A23C12CA72', '\xddde36a58382061e8fc66a575a3281ca15c91f457282e536bc6ba6ee08d16dbb', true);
INSERT INTO public.machine VALUES (7, '2025-10-31 09:49:03.839879', 'agent-pdns', 8891, '{"Os": "linux", "Cpus": 6, "HostID": "7e916d04-3bed-ca44-b334-3e2f68ba44ff", "Memory": 7, "Uptime": 4, "CpusLoad": "0.12 0.33 0.53", "Hostname": "agent-pdns", "Platform": "debian", "KernelArch": "aarch64", "UsedMemory": 27, "AgentVersion": "2.3.0", "KernelVersion": "6.8.0-50-generic", "PlatformFamily": "debian", "PlatformVersion": "12.11", "VirtualizationRole": "guest", "VirtualizationSystem": "docker"}', '2025-10-31 10:00:37.798189', NULL, 'A8EDE61A67982C9A1D8F1B7EF34A6038A173352D3C74C424022734BB2A275B60', '\xdf0a2484dee6a1d840294e8d026a69796cc3abea3fa569ac490ca73d7f579dad', true);
INSERT INTO public.machine VALUES (5, '2025-10-31 09:49:01.964857', 'agent-kea-ha1', 8886, '{"Os": "linux", "Cpus": 6, "HostID": "7e916d04-3bed-ca44-b334-3e2f68ba44ff", "Memory": 7, "Uptime": 4, "CpusLoad": "0.12 0.33 0.53", "Hostname": "agent-kea-ha1", "Platform": "debian", "KernelArch": "aarch64", "UsedMemory": 27, "AgentVersion": "2.3.0", "KernelVersion": "6.8.0-50-generic", "PlatformFamily": "debian", "PlatformVersion": "12.11", "VirtualizationRole": "guest", "VirtualizationSystem": "docker"}', '2025-10-31 10:00:37.802984', NULL, 'B5842FF9274A9878AF0D90BEFEECBA051DEC703799F2302EC7E72EC369E865A0', '\x1a6a18b40d25f712b0e5ee013dc43845730f33b4fbf36d1fcd0399ffeadb1e74', true);
INSERT INTO public.machine VALUES (9, '2025-10-31 09:49:04.489664', 'agent-bind9', 8883, '{"Os": "linux", "Cpus": 6, "HostID": "9a4b5364-e2cb-49bd-aa5a-0ab35c719534", "Memory": 7, "Uptime": 4, "CpusLoad": "0.12 0.33 0.53", "Hostname": "agent-bind9", "Platform": "alpine", "KernelArch": "x86_64", "UsedMemory": 27, "AgentVersion": "2.3.0", "KernelVersion": "6.8.0-50-generic", "PlatformFamily": "alpine", "PlatformVersion": "3.22.2", "VirtualizationRole": "guest", "VirtualizationSystem": "docker"}', '2025-10-31 10:00:37.819333', NULL, '61608F9B460BFD2D8BB97DDDDC13864194A924A912C08E93FFEF011334893B32', '\x45813ec87d03ed2a27aa49e5a37470a7b7c3f172b284fbfb4009a563ee56c635', true);
INSERT INTO public.machine VALUES (2, '2025-10-31 09:49:01.725593', 'agent-kea-ha3', 8890, '{"Os": "linux", "Cpus": 6, "HostID": "7e916d04-3bed-ca44-b334-3e2f68ba44ff", "Memory": 7, "Uptime": 4, "CpusLoad": "0.12 0.33 0.53", "Hostname": "agent-kea-ha3", "Platform": "debian", "KernelArch": "aarch64", "UsedMemory": 27, "AgentVersion": "2.3.0", "KernelVersion": "6.8.0-50-generic", "PlatformFamily": "debian", "PlatformVersion": "12.11", "VirtualizationRole": "guest", "VirtualizationSystem": "docker"}', '2025-10-31 10:00:37.754159', NULL, '8FDC4BD32BC24A7AF5B01AFAE2F976C84347463DEA9100BE5DFE7DA26971C9D9', '\x714d0a149fbc7ddb1872740ae78aa3ffff4baf30285556305640054c3cfc9038', true);
INSERT INTO public.machine VALUES (3, '2025-10-31 09:49:01.817658', 'agent-kea-ha2', 8885, '{"Os": "linux", "Cpus": 6, "HostID": "7e916d04-3bed-ca44-b334-3e2f68ba44ff", "Memory": 7, "Uptime": 4, "CpusLoad": "0.12 0.33 0.53", "Hostname": "agent-kea-ha2", "Platform": "debian", "KernelArch": "aarch64", "UsedMemory": 27, "AgentVersion": "2.3.0", "KernelVersion": "6.8.0-50-generic", "PlatformFamily": "debian", "PlatformVersion": "12.11", "VirtualizationRole": "guest", "VirtualizationSystem": "docker"}', '2025-10-31 10:00:37.895369', NULL, 'C09ECF24B4E9E994B4CE7D158CAB10738E36DF7B52DC15BFB8AD7B88F94DCC48', '\xe614dee0d516ce267c689d5c9ee2646e58cac46767b394166b9a15b6cbde140c', true);
INSERT INTO public.machine VALUES (4, '2025-10-31 09:49:01.936344', 'agent-kea', 8888, '{"Os": "linux", "Cpus": 6, "HostID": "7e916d04-3bed-ca44-b334-3e2f68ba44ff", "Memory": 7, "Uptime": 4, "CpusLoad": "0.12 0.33 0.53", "Hostname": "agent-kea", "Platform": "debian", "KernelArch": "aarch64", "UsedMemory": 27, "AgentVersion": "2.3.0", "KernelVersion": "6.8.0-50-generic", "PlatformFamily": "debian", "PlatformVersion": "12.11", "VirtualizationRole": "guest", "VirtualizationSystem": "docker"}', '2025-10-31 10:00:37.912335', NULL, '9F7D7A802422283D601E7EAA29E2EE27939F131A499A1E51F03BA09626646251', '\x27dce21743e8da71d166db95cb43c80ca74c45d0c19dc17f313c3a34209ebbc7', true);
INSERT INTO public.machine VALUES (8, '2025-10-31 09:49:04.44542', 'agent-bind9-2', 8882, '{"Os": "linux", "Cpus": 6, "HostID": "9a4b5364-e2cb-49bd-aa5a-0ab35c719534", "Memory": 7, "Uptime": 4, "CpusLoad": "0.12 0.33 0.53", "Hostname": "agent-bind9-2", "Platform": "alpine", "KernelArch": "x86_64", "UsedMemory": 27, "AgentVersion": "2.3.0", "KernelVersion": "6.8.0-50-generic", "PlatformFamily": "alpine", "PlatformVersion": "3.22.2", "VirtualizationRole": "guest", "VirtualizationSystem": "docker"}', '2025-10-31 10:00:37.928721', NULL, 'B42973BB08E797E5F9E9D6533957F492AE55F12745A39A4ED2564AB2DD8C8509', '\x281cf7034b136ddb07ed2038ec026230c522faaf5c84c5dc641a6d47c354fcc8', true);


--
-- TOC entry 3942 (class 0 OID 17211)
-- Dependencies: 289
-- Data for Name: pdns_daemon; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.pdns_daemon VALUES (1, 19, '{"URL": "/api/v1/servers/localhost", "ZonesURL": "/api/v1/servers/localhost/zones{/zone}", "ConfigURL": "/api/v1/servers/localhost/config{/config_setting}", "AutoprimariesURL": "/api/v1/servers/localhost/autoprimaries{/autoprimary}"}');


--
-- TOC entry 3895 (class 0 OID 16628)
-- Dependencies: 242
-- Data for Name: prefix_pool; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.prefix_pool VALUES (2, '2025-10-31 09:53:45.426795', '3001:db8:9::/56', 64, '3001:db8:9:0:ca00::/72', '{"pool-id": 9}', NULL, NULL, 19, '{"total-pds": "256", "assigned-pds": "0", "reclaimed-leases": "0", "cumulative-assigned-pds": "0"}', '2025-10-31 09:59:57.038192', 0);
INSERT INTO public.prefix_pool VALUES (1, '2025-10-31 09:53:45.426795', '3001:db8:8::/56', 64, NULL, '{"pool-id": 8}', NULL, NULL, 19, '{"total-pds": "256", "assigned-pds": "0", "reclaimed-leases": "0", "cumulative-assigned-pds": "0"}', '2025-10-31 09:59:57.038402', 0);


--
-- TOC entry 3919 (class 0 OID 16924)
-- Dependencies: 266
-- Data for Name: rps_interval; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.rps_interval VALUES (3, '2025-10-31 09:53:56.532753', 60, NULL);
INSERT INTO public.rps_interval VALUES (5, '2025-10-31 09:53:56.552901', 60, NULL);
INSERT INTO public.rps_interval VALUES (9, '2025-10-31 09:53:56.560549', 60, NULL);
INSERT INTO public.rps_interval VALUES (13, '2025-10-31 09:53:56.570967', 60, NULL);
INSERT INTO public.rps_interval VALUES (15, '2025-10-31 09:53:56.579704', 60, NULL);
INSERT INTO public.rps_interval VALUES (3, '2025-10-31 09:54:56.634123', 60, NULL);
INSERT INTO public.rps_interval VALUES (5, '2025-10-31 09:54:56.661312', 60, NULL);
INSERT INTO public.rps_interval VALUES (9, '2025-10-31 09:54:56.67233', 60, NULL);
INSERT INTO public.rps_interval VALUES (13, '2025-10-31 09:54:56.680873', 60, NULL);
INSERT INTO public.rps_interval VALUES (15, '2025-10-31 09:54:56.689739', 60, NULL);
INSERT INTO public.rps_interval VALUES (3, '2025-10-31 09:55:56.741018', 60, NULL);
INSERT INTO public.rps_interval VALUES (5, '2025-10-31 09:55:56.756089', 60, NULL);
INSERT INTO public.rps_interval VALUES (9, '2025-10-31 09:55:56.762412', 60, NULL);
INSERT INTO public.rps_interval VALUES (13, '2025-10-31 09:55:56.769933', 60, NULL);
INSERT INTO public.rps_interval VALUES (15, '2025-10-31 09:55:56.775986', 60, NULL);
INSERT INTO public.rps_interval VALUES (3, '2025-10-31 09:56:56.822722', 60, NULL);
INSERT INTO public.rps_interval VALUES (5, '2025-10-31 09:56:56.836875', 60, NULL);
INSERT INTO public.rps_interval VALUES (9, '2025-10-31 09:56:56.841965', 60, NULL);
INSERT INTO public.rps_interval VALUES (13, '2025-10-31 09:56:56.847364', 60, NULL);
INSERT INTO public.rps_interval VALUES (15, '2025-10-31 09:56:56.852024', 60, NULL);
INSERT INTO public.rps_interval VALUES (3, '2025-10-31 09:57:56.887517', 60, NULL);
INSERT INTO public.rps_interval VALUES (5, '2025-10-31 09:57:56.900192', 60, NULL);
INSERT INTO public.rps_interval VALUES (9, '2025-10-31 09:57:56.905283', 60, NULL);
INSERT INTO public.rps_interval VALUES (13, '2025-10-31 09:57:56.910379', 60, NULL);
INSERT INTO public.rps_interval VALUES (15, '2025-10-31 09:57:56.915084', 60, NULL);
INSERT INTO public.rps_interval VALUES (3, '2025-10-31 09:58:56.954463', 61, NULL);
INSERT INTO public.rps_interval VALUES (5, '2025-10-31 09:58:56.969144', 61, NULL);
INSERT INTO public.rps_interval VALUES (9, '2025-10-31 09:58:56.974892', 61, NULL);
INSERT INTO public.rps_interval VALUES (13, '2025-10-31 09:58:56.980393', 61, NULL);
INSERT INTO public.rps_interval VALUES (15, '2025-10-31 09:58:56.985047', 61, NULL);


--
-- TOC entry 3928 (class 0 OID 17028)
-- Dependencies: 275
-- Data for Name: scheduled_config_change; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- TOC entry 3921 (class 0 OID 16931)
-- Dependencies: 268
-- Data for Name: secret; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.secret VALUES ('cakey', '-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgR3AkJRd3lDi2OfTH
AtOX+5oazMAun/+WIbk0Hpi7xUGhRANCAAQuw/DbYj/CEjqlh8NuCOR+NvVfW28B
POipsfbFNtwnUnHwtB0yikLkuGx/vN294IcJesdYSGKUe8z9O7QXVcQY
-----END PRIVATE KEY-----
');
INSERT INTO public.secret VALUES ('cacert', '-----BEGIN CERTIFICATE-----
MIIBvDCCAWGgAwIBAgIBATAKBggqhkjOPQQDAjAzMQswCQYDVQQGEwJVUzESMBAG
A1UEChMJSVNDIFN0b3JrMRAwDgYDVQQDEwdSb290IENBMCAXDTI1MTAzMTA5NDg1
NloYDzIwNTUxMDMxMDk0ODU2WjAzMQswCQYDVQQGEwJVUzESMBAGA1UEChMJSVND
IFN0b3JrMRAwDgYDVQQDEwdSb290IENBMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcD
QgAELsPw22I/whI6pYfDbgjkfjb1X1tvATzoqbH2xTbcJ1Jx8LQdMopC5Lhsf7zd
veCHCXrHWEhilHvM/Tu0F1XEGKNkMGIwDgYDVR0PAQH/BAQDAgKEMB0GA1UdJQQW
MBQGCCsGAQUFBwMCBggrBgEFBQcDATASBgNVHRMBAf8ECDAGAQH/AgEBMB0GA1Ud
DgQWBBRRq1UqPJuOnhB7NfZLOl0tL2XtgDAKBggqhkjOPQQDAgNJADBGAiEArLEQ
Nrtm98AerCNXSQgd6T2bBHU/maCtFhvxg2/hoj0CIQD4DK8Q4FAqdZqDwWbxT5oL
9RtG6hScsbXOpaTEl+vd5w==
-----END CERTIFICATE-----
');
INSERT INTO public.secret VALUES ('srvkey', '-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgQ77v/beTAuW6IrdV
xaXgnNBxJdWss/p6b2M/YA5UHk6hRANCAAQA702atME8yOrBcn8n5dLjOcTyWcUl
iZCofmJDod72ok08Z8EHaK4Ska/gB79IvSr04loRxK/J+/3m4axlfSov
-----END PRIVATE KEY-----
');
INSERT INTO public.secret VALUES ('srvcert', '-----BEGIN CERTIFICATE-----
MIICQzCCAemgAwIBAgIBAjAKBggqhkjOPQQDAjAzMQswCQYDVQQGEwJVUzESMBAG
A1UEChMJSVNDIFN0b3JrMRAwDgYDVQQDEwdSb290IENBMCAXDTI1MTAzMTA5NDg1
NloYDzIwNTUxMDMxMDk0ODU2WjBGMQswCQYDVQQGEwJVUzESMBAGA1UEChMJSVND
IFN0b3JrMQ8wDQYDVQQLEwZzZXJ2ZXIxEjAQBgNVBAMTCWxvY2FsaG9zdDBZMBMG
ByqGSM49AgEGCCqGSM49AwEHA0IABADvTZq0wTzI6sFyfyfl0uM5xPJZxSWJkKh+
YkOh3vaiTTxnwQdorhKRr+AHv0i9KvTiWhHEr8n7/ebhrGV9Ki+jgdgwgdUwEwYD
VR0lBAwwCgYIKwYBBQUHAwIwHwYDVR0jBBgwFoAUUatVKjybjp4QezX2SzpdLS9l
7YAwgZwGA1UdEQSBlDCBkYIJbG9jYWxob3N0ggxjZWFmYzE3ZjAxMDaCCWxvY2Fs
aG9zdIINaXA2LWxvY2FsaG9zdIIMaXA2LWxvb3BiYWNrggxjZWFmYzE3ZjAxMDaH
BH8AAAGHBKwYAAaHEAAAAAAAAAAAAAAAAAAAAAGHEDAJDbgAAQAAAAAAAAAAABCH
EP6AAAAAAAAAAEKs//4YAAYwCgYIKoZIzj0EAwIDSAAwRQIgHxbFba1NBXuBs/ZT
M7MrpDQCHjyKCOOmKmsrfP3X7KQCIQDJfE/oHZzlWNonN+zuAiEWz47iisU8eRzK
kenn415T2g==
-----END CERTIFICATE-----
');
INSERT INTO public.secret VALUES ('srvtkn', 'UiSE9qiAghmVBXL3ZLzFnFgXXuflKeq7');


--
-- TOC entry 3884 (class 0 OID 16527)
-- Dependencies: 231
-- Data for Name: service; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.service VALUES (2, 'service-0000000002', '2025-10-31 09:53:46.547234');
INSERT INTO public.service VALUES (1, 'service-0000000001', '2025-10-31 09:53:45.842424');


--
-- TOC entry 3871 (class 0 OID 16390)
-- Dependencies: 218
-- Data for Name: sessions; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- TOC entry 3898 (class 0 OID 16701)
-- Dependencies: 245
-- Data for Name: setting; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.setting VALUES ('bind9_stats_puller_interval', 1, '60');
INSERT INTO public.setting VALUES ('kea_stats_puller_interval', 1, '60');
INSERT INTO public.setting VALUES ('kea_hosts_puller_interval', 1, '60');
INSERT INTO public.setting VALUES ('kea_status_puller_interval', 1, '30');
INSERT INTO public.setting VALUES ('apps_state_puller_interval', 1, '30');
INSERT INTO public.setting VALUES ('grafana_url', 3, '');
INSERT INTO public.setting VALUES ('grafana_dhcp4_dashboard_id', 3, 'hRf18FvWz');
INSERT INTO public.setting VALUES ('grafana_dhcp6_dashboard_id', 3, 'AQPHKJUGz');
INSERT INTO public.setting VALUES ('enable_machine_registration', 2, 'true');
INSERT INTO public.setting VALUES ('enable_online_software_versions', 2, 'true');


--
-- TOC entry 3889 (class 0 OID 16585)
-- Dependencies: 236
-- Data for Name: shared_network; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.shared_network VALUES (1, '2025-10-31 09:53:44.651209', 'frog', 4, 0, 0, '{"total-nas": "2", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "total-out-of-pool-nas": "-411", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0"}', '2025-10-31 09:59:57.069981', 0, NULL);
INSERT INTO public.shared_network VALUES (2, '2025-10-31 09:53:44.651209', 'mouse', 4, 0, 0, '{"total-nas": "0", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "total-out-of-pool-nas": "-445", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0"}', '2025-10-31 09:59:57.070191', 0, NULL);
INSERT INTO public.shared_network VALUES (3, '2025-10-31 09:53:45.426795', 'frog', 6, 0, 0, '{"total-nas": "-221361210359491330048", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "total-out-of-pool-nas": "-221361210359491330048", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0"}', '2025-10-31 09:59:57.070364', 0, NULL);
INSERT INTO public.shared_network VALUES (4, '2025-10-31 09:53:45.842424', 'esperanto', 4, 0, 0, '{"total-nas": "10", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "total-out-of-pool-nas": "10", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0"}', '2025-10-31 09:59:57.070546', 0, NULL);


--
-- TOC entry 3906 (class 0 OID 16795)
-- Dependencies: 253
-- Data for Name: statistic; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.statistic VALUES ('total-nas', 221362336259398172681);
INSERT INTO public.statistic VALUES ('total-pds', 514);
INSERT INTO public.statistic VALUES ('total-addresses', 1476);
INSERT INTO public.statistic VALUES ('assigned-addresses', 2);
INSERT INTO public.statistic VALUES ('declined-nas', 1);
INSERT INTO public.statistic VALUES ('assigned-pds', 1);
INSERT INTO public.statistic VALUES ('declined-addresses', 1);
INSERT INTO public.statistic VALUES ('assigned-nas', 2);


--
-- TOC entry 3891 (class 0 OID 16595)
-- Dependencies: 238
-- Data for Name: subnet; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.subnet VALUES (10, '2025-10-31 09:53:44.651209', '192.1.17.0/24', 2, NULL, 0, 0, '{"total-addresses": "245", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.066366', 0, 0);
INSERT INTO public.subnet VALUES (6, '2025-10-31 09:53:44.651209', '192.0.10.0/24', 1, NULL, 0, 0, '{"total-addresses": "50", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.065044', 0, 0);
INSERT INTO public.subnet VALUES (17, '2025-10-31 09:53:45.426795', '5004::/16', 3, NULL, 0, 0, '{"total-nas": "0", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "total-out-of-pool-nas": "0", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.067738', 0, 0);
INSERT INTO public.subnet VALUES (15, '2025-10-31 09:53:45.426795', '5002::/16', 3, NULL, 0, 0, '{"total-nas": "0", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "total-out-of-pool-nas": "0", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.067949', 0, 0);
INSERT INTO public.subnet VALUES (2, '2025-10-31 09:53:44.651209', '192.0.6.0/24', 1, NULL, 0, 0, '{"total-addresses": "110", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.065305', 0, 0);
INSERT INTO public.subnet VALUES (4, '2025-10-31 09:53:44.651209', '192.0.8.0/24', 1, NULL, 0, 0, '{"total-addresses": "50", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.065542', 0, 0);
INSERT INTO public.subnet VALUES (18, '2025-10-31 09:53:45.426795', '5005::/16', 3, NULL, 0, 0, '{"total-nas": "0", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "total-out-of-pool-nas": "0", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.068141', 0, 0);
INSERT INTO public.subnet VALUES (22, '2025-10-31 09:53:45.842424', '192.110.111.0/24', 4, NULL, 0, 0, '{"total-addresses": "10", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "10", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.068537', 0, 0);
INSERT INTO public.subnet VALUES (9, '2025-10-31 09:53:44.651209', '192.1.16.0/24', 2, NULL, 0, 0, '{"total-addresses": "150", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.066741', 0, 0);
INSERT INTO public.subnet VALUES (16, '2025-10-31 09:53:45.426795', '5003::/16', 3, NULL, 0, 0, '{"total-nas": "0", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "total-out-of-pool-nas": "0", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.068337', 0, 0);
INSERT INTO public.subnet VALUES (3, '2025-10-31 09:53:44.651209', '192.0.7.0/24', 1, NULL, 0, 0, '{"total-addresses": "100", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.065796', 0, 0);
INSERT INTO public.subnet VALUES (12, '2025-10-31 09:53:45.426795', '4001:db8:1::/64', 3, NULL, 0, 0, '{"total-nas": "281474976710656", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "total-out-of-pool-nas": "0", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.06703', 0, 0);
INSERT INTO public.subnet VALUES (1, '2025-10-31 09:53:44.651209', '192.0.5.0/24', 1, NULL, 0, 0, '{"total-addresses": "52", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "2", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.06394', 0, 0);
INSERT INTO public.subnet VALUES (7, '2025-10-31 09:53:44.651209', '192.0.10.80/29', 1, NULL, 0, 0, '{"total-addresses": "3", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.064478', 0, 0);
INSERT INTO public.subnet VALUES (5, '2025-10-31 09:53:44.651209', '192.0.9.0/24', 1, NULL, 0, 0, '{"total-addresses": "50", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.064765', 0, 0);
INSERT INTO public.subnet VALUES (8, '2025-10-31 09:53:44.651209', '192.1.15.0/24', 2, NULL, 0, 0, '{"total-addresses": "50", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.066025', 0, 0);
INSERT INTO public.subnet VALUES (13, '2025-10-31 09:53:45.426795', '5000::/16', 3, NULL, 0, 0, '{"total-nas": "0", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "total-out-of-pool-nas": "0", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.067282', 0, 0);
INSERT INTO public.subnet VALUES (14, '2025-10-31 09:53:45.426795', '5001::/16', 3, NULL, 0, 0, '{"total-nas": "0", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "total-out-of-pool-nas": "0", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.067491', 0, 0);
INSERT INTO public.subnet VALUES (23, '2025-10-31 09:53:45.842424', '192.110.112.0/24', 4, NULL, 0, 0, '{"total-addresses": "0", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.068735', 0, 0);
INSERT INTO public.subnet VALUES (11, '2025-10-31 09:53:44.651209', '192.0.2.0/24', NULL, NULL, 10, 0, '{"total-addresses": "201", "assigned-addresses": "2", "declined-addresses": "1", "total-out-of-pool-addresses": "1", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.06892', 0, 0);
INSERT INTO public.subnet VALUES (25, '2025-10-31 09:53:46.547234', '192.0.3.0/24', NULL, NULL, 0, 0, '{"total-addresses": "200", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.069115', 0, 0);
INSERT INTO public.subnet VALUES (21, '2025-10-31 09:53:45.426795', '3001:1234:5678:90ab:cdef:1f2e:3d4c:5b68/125', NULL, NULL, 0, 0, '{"total-nas": "4", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "total-out-of-pool-nas": "0", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.069293', 0, 0);
INSERT INTO public.subnet VALUES (19, '2025-10-31 09:53:45.426795', '3001:db8:1::/64', NULL, NULL, 0, 2, '{"total-nas": "844424930131972", "total-pds": "513", "assigned-nas": "2", "assigned-pds": "1", "declined-nas": "1", "total-out-of-pool-nas": "4", "total-out-of-pool-pds": "1", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "1", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.069468', 0, 1000);
INSERT INTO public.subnet VALUES (20, '2025-10-31 09:53:45.426795', '3000:db8:1::/64', NULL, NULL, 0, 0, '{"total-nas": "281474976710656", "total-pds": "0", "assigned-nas": "0", "assigned-pds": "0", "declined-nas": "0", "total-out-of-pool-nas": "0", "total-out-of-pool-pds": "0", "assigned-out-of-pool-nas": "0", "assigned-out-of-pool-pds": "0", "declined-out-of-pool-nas": "0"}', '2025-10-31 09:59:57.069647', 0, 0);
INSERT INTO public.subnet VALUES (24, '2025-10-31 09:53:45.842424', '192.0.20.0/24', NULL, NULL, 0, 0, '{"total-addresses": "200", "assigned-addresses": "0", "declined-addresses": "0", "total-out-of-pool-addresses": "0", "assigned-out-of-pool-addresses": "0", "declined-out-of-pool-addresses": "0"}', '2025-10-31 09:59:57.069822', 0, 0);


--
-- TOC entry 3879 (class 0 OID 16485)
-- Dependencies: 226
-- Data for Name: system_group; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.system_group VALUES (1, 'super-admin', 'This group of users can access all system components.');
INSERT INTO public.system_group VALUES (2, 'admin', 'This group of users can do everything except manage user accounts.');
INSERT INTO public.system_group VALUES (3, 'read-only', 'This group of users can only have read access to system components and APIs. Users that belong to this group cannot perform Create, Update nor Delete actions.');


--
-- TOC entry 3872 (class 0 OID 16395)
-- Dependencies: 219
-- Data for Name: system_user; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public."system_user" VALUES (1, NULL, 'admin', 'admin', 'admin', 'internal', NULL, true);
INSERT INTO public."system_user" VALUES (2, 'admin@example.com', 'Admin', 'Admin', 'admin', 'ldap', 'cn=admin,ou=users,dc=example,dc=org', false);


--
-- TOC entry 3933 (class 0 OID 17110)
-- Dependencies: 280
-- Data for Name: system_user_password; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.system_user_password VALUES (1, '$2a$06$PmNL9Ryd7ecLfBDVIPgC.OkTarv5LopTHdG3uxGkDbhxXISwMdw.a');


--
-- TOC entry 3882 (class 0 OID 16495)
-- Dependencies: 229
-- Data for Name: system_user_to_group; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.system_user_to_group VALUES (1, 1);
INSERT INTO public.system_user_to_group VALUES (2, 1);


--
-- TOC entry 3936 (class 0 OID 17155)
-- Dependencies: 283
-- Data for Name: zone; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.zone VALUES (1, '0.0.10.in-addr.arpa', 'arpa.in-addr.10.0.0');
INSERT INTO public.zone VALUES (2, '1.0.10.in-addr.arpa', 'arpa.in-addr.10.0.1');
INSERT INTO public.zone VALUES (3, '2.0.10.in-addr.arpa', 'arpa.in-addr.10.0.2');
INSERT INTO public.zone VALUES (4, '3.0.10.in-addr.arpa', 'arpa.in-addr.10.0.3');
INSERT INTO public.zone VALUES (5, '0.16.172.in-addr.arpa', 'arpa.in-addr.172.16.0');
INSERT INTO public.zone VALUES (6, '1.16.172.in-addr.arpa', 'arpa.in-addr.172.16.1');
INSERT INTO public.zone VALUES (7, '2.16.172.in-addr.arpa', 'arpa.in-addr.172.16.2');
INSERT INTO public.zone VALUES (8, '3.16.172.in-addr.arpa', 'arpa.in-addr.172.16.3');
INSERT INTO public.zone VALUES (9, 'pdns.example.com', 'com.example.pdns');
INSERT INTO public.zone VALUES (10, 'pdns.example.org', 'org.example.pdns');
INSERT INTO public.zone VALUES (11, 'authors.bind', 'bind.authors');
INSERT INTO public.zone VALUES (15, 'EMPTY.AS112.ARPA', 'ARPA.AS112.EMPTY');
INSERT INTO public.zone VALUES (16, 'HOME.ARPA', 'ARPA.HOME');
INSERT INTO public.zone VALUES (17, '0.IN-ADDR.ARPA', 'ARPA.IN-ADDR.0');
INSERT INTO public.zone VALUES (18, '10.IN-ADDR.ARPA', 'ARPA.IN-ADDR.10');
INSERT INTO public.zone VALUES (19, '100.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.100');
INSERT INTO public.zone VALUES (20, '101.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.101');
INSERT INTO public.zone VALUES (21, '102.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.102');
INSERT INTO public.zone VALUES (22, '103.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.103');
INSERT INTO public.zone VALUES (23, '104.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.104');
INSERT INTO public.zone VALUES (24, '105.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.105');
INSERT INTO public.zone VALUES (25, '106.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.106');
INSERT INTO public.zone VALUES (26, '107.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.107');
INSERT INTO public.zone VALUES (217, 'bind9.example.com', 'com.example.bind9');
INSERT INTO public.zone VALUES (104, '113.0.203.IN-ADDR.ARPA', 'ARPA.IN-ADDR.203.0.113');
INSERT INTO public.zone VALUES (105, '255.255.255.255.IN-ADDR.ARPA', 'ARPA.IN-ADDR.255.255.255.255');
INSERT INTO public.zone VALUES (106, '0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.IP6.ARPA', 'ARPA.IP6.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0');
INSERT INTO public.zone VALUES (107, '1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.IP6.ARPA', 'ARPA.IP6.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.1');
INSERT INTO public.zone VALUES (108, '8.B.D.0.1.0.0.2.IP6.ARPA', 'ARPA.IP6.2.0.0.1.0.D.B.8');
INSERT INTO public.zone VALUES (109, 'D.F.IP6.ARPA', 'ARPA.IP6.F.D');
INSERT INTO public.zone VALUES (110, '8.E.F.IP6.ARPA', 'ARPA.IP6.F.E.8');
INSERT INTO public.zone VALUES (111, '9.E.F.IP6.ARPA', 'ARPA.IP6.F.E.9');
INSERT INTO public.zone VALUES (112, 'A.E.F.IP6.ARPA', 'ARPA.IP6.F.E.A');
INSERT INTO public.zone VALUES (113, 'B.E.F.IP6.ARPA', 'ARPA.IP6.F.E.B');
INSERT INTO public.zone VALUES (114, 'RESOLVER.ARPA', 'ARPA.RESOLVER');
INSERT INTO public.zone VALUES (218, 'drop.rpz.example.com', 'com.example.rpz.drop');
INSERT INTO public.zone VALUES (12, 'hostname.bind', 'bind.hostname');
INSERT INTO public.zone VALUES (13, 'version.bind', 'bind.version');
INSERT INTO public.zone VALUES (14, 'id.server', 'server.id');
INSERT INTO public.zone VALUES (222, '.', '.');
INSERT INTO public.zone VALUES (27, '108.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.108');
INSERT INTO public.zone VALUES (28, '109.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.109');
INSERT INTO public.zone VALUES (29, '110.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.110');
INSERT INTO public.zone VALUES (30, '111.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.111');
INSERT INTO public.zone VALUES (31, '112.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.112');
INSERT INTO public.zone VALUES (32, '113.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.113');
INSERT INTO public.zone VALUES (33, '114.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.114');
INSERT INTO public.zone VALUES (34, '115.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.115');
INSERT INTO public.zone VALUES (35, '116.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.116');
INSERT INTO public.zone VALUES (36, '117.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.117');
INSERT INTO public.zone VALUES (37, '118.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.118');
INSERT INTO public.zone VALUES (38, '119.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.119');
INSERT INTO public.zone VALUES (39, '120.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.120');
INSERT INTO public.zone VALUES (40, '121.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.121');
INSERT INTO public.zone VALUES (41, '122.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.122');
INSERT INTO public.zone VALUES (42, '123.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.123');
INSERT INTO public.zone VALUES (43, '124.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.124');
INSERT INTO public.zone VALUES (44, '125.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.125');
INSERT INTO public.zone VALUES (45, '126.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.126');
INSERT INTO public.zone VALUES (46, '127.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.127');
INSERT INTO public.zone VALUES (47, '64.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.64');
INSERT INTO public.zone VALUES (48, '65.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.65');
INSERT INTO public.zone VALUES (49, '66.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.66');
INSERT INTO public.zone VALUES (50, '67.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.67');
INSERT INTO public.zone VALUES (51, '68.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.68');
INSERT INTO public.zone VALUES (52, '69.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.69');
INSERT INTO public.zone VALUES (53, '70.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.70');
INSERT INTO public.zone VALUES (54, '71.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.71');
INSERT INTO public.zone VALUES (55, '72.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.72');
INSERT INTO public.zone VALUES (56, '73.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.73');
INSERT INTO public.zone VALUES (57, '74.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.74');
INSERT INTO public.zone VALUES (58, '75.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.75');
INSERT INTO public.zone VALUES (59, '76.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.76');
INSERT INTO public.zone VALUES (60, '77.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.77');
INSERT INTO public.zone VALUES (61, '78.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.78');
INSERT INTO public.zone VALUES (62, '79.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.79');
INSERT INTO public.zone VALUES (63, '80.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.80');
INSERT INTO public.zone VALUES (64, '81.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.81');
INSERT INTO public.zone VALUES (65, '82.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.82');
INSERT INTO public.zone VALUES (66, '83.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.83');
INSERT INTO public.zone VALUES (67, '84.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.84');
INSERT INTO public.zone VALUES (68, '85.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.85');
INSERT INTO public.zone VALUES (69, '86.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.86');
INSERT INTO public.zone VALUES (70, '87.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.87');
INSERT INTO public.zone VALUES (71, '88.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.88');
INSERT INTO public.zone VALUES (72, '89.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.89');
INSERT INTO public.zone VALUES (73, '90.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.90');
INSERT INTO public.zone VALUES (74, '91.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.91');
INSERT INTO public.zone VALUES (75, '92.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.92');
INSERT INTO public.zone VALUES (76, '93.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.93');
INSERT INTO public.zone VALUES (77, '94.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.94');
INSERT INTO public.zone VALUES (78, '95.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.95');
INSERT INTO public.zone VALUES (79, '96.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.96');
INSERT INTO public.zone VALUES (80, '97.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.97');
INSERT INTO public.zone VALUES (81, '98.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.98');
INSERT INTO public.zone VALUES (82, '99.100.IN-ADDR.ARPA', 'ARPA.IN-ADDR.100.99');
INSERT INTO public.zone VALUES (83, '127.IN-ADDR.ARPA', 'ARPA.IN-ADDR.127');
INSERT INTO public.zone VALUES (84, '254.169.IN-ADDR.ARPA', 'ARPA.IN-ADDR.169.254');
INSERT INTO public.zone VALUES (85, '16.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.16');
INSERT INTO public.zone VALUES (86, '17.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.17');
INSERT INTO public.zone VALUES (87, '18.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.18');
INSERT INTO public.zone VALUES (88, '19.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.19');
INSERT INTO public.zone VALUES (89, '20.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.20');
INSERT INTO public.zone VALUES (90, '21.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.21');
INSERT INTO public.zone VALUES (91, '22.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.22');
INSERT INTO public.zone VALUES (92, '23.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.23');
INSERT INTO public.zone VALUES (93, '24.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.24');
INSERT INTO public.zone VALUES (94, '25.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.25');
INSERT INTO public.zone VALUES (95, '26.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.26');
INSERT INTO public.zone VALUES (96, '27.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.27');
INSERT INTO public.zone VALUES (97, '28.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.28');
INSERT INTO public.zone VALUES (98, '29.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.29');
INSERT INTO public.zone VALUES (99, '30.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.30');
INSERT INTO public.zone VALUES (100, '31.172.IN-ADDR.ARPA', 'ARPA.IN-ADDR.172.31');
INSERT INTO public.zone VALUES (101, '2.0.192.IN-ADDR.ARPA', 'ARPA.IN-ADDR.192.0.2');
INSERT INTO public.zone VALUES (102, '168.192.IN-ADDR.ARPA', 'ARPA.IN-ADDR.192.168');
INSERT INTO public.zone VALUES (103, '100.51.198.IN-ADDR.ARPA', 'ARPA.IN-ADDR.198.51.100');
INSERT INTO public.zone VALUES (324, 'rpz.local', 'local.rpz');
INSERT INTO public.zone VALUES (115, 'bind9.example.org', 'org.example.bind9');
INSERT INTO public.zone VALUES (326, 'test', 'test');


--
-- TOC entry 3940 (class 0 OID 17190)
-- Dependencies: 287
-- Data for Name: zone_inventory_state; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.zone_inventory_state VALUES (1, 19, '2025-10-31 09:54:18.289374', '{"Error": null, "Status": "ok", "ZoneCount": 10, "BuiltinZoneCount": 0, "DistinctZoneCount": 10}');
INSERT INTO public.zone_inventory_state VALUES (2, 20, '2025-10-31 09:54:18.30285', '{"Error": null, "Status": "ok", "ZoneCount": 207, "BuiltinZoneCount": 104, "DistinctZoneCount": 107}');
INSERT INTO public.zone_inventory_state VALUES (3, 21, '2025-10-31 09:54:18.318823', '{"Error": null, "Status": "ok", "ZoneCount": 109, "BuiltinZoneCount": 104, "DistinctZoneCount": 109}');


--
-- TOC entry 3986 (class 0 OID 0)
-- Dependencies: 239
-- Name: address_pool_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.address_pool_id_seq', 57, true);


--
-- TOC entry 3987 (class 0 OID 0)
-- Dependencies: 223
-- Name: app_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.app_id_seq', 8, true);


--
-- TOC entry 3988 (class 0 OID 0)
-- Dependencies: 260
-- Name: bind9_daemon_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.bind9_daemon_id_seq', 2, true);


--
-- TOC entry 3989 (class 0 OID 0)
-- Dependencies: 267
-- Name: certs_serial_number_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.certs_serial_number_seq', 11, true);


--
-- TOC entry 3990 (class 0 OID 0)
-- Dependencies: 276
-- Name: config_checker_preference_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.config_checker_preference_id_seq', 1, false);


--
-- TOC entry 3991 (class 0 OID 0)
-- Dependencies: 269
-- Name: config_report_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.config_report_id_seq', 114, true);


--
-- TOC entry 3992 (class 0 OID 0)
-- Dependencies: 272
-- Name: config_review_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.config_review_id_seq', 13, true);


--
-- TOC entry 3993 (class 0 OID 0)
-- Dependencies: 254
-- Name: daemon_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.daemon_id_seq', 21, true);


--
-- TOC entry 3994 (class 0 OID 0)
-- Dependencies: 262
-- Name: event_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.event_id_seq', 57, true);


--
-- TOC entry 3995 (class 0 OID 0)
-- Dependencies: 216
-- Name: gopg_migrations_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.gopg_migrations_id_seq', 70, true);


--
-- TOC entry 3996 (class 0 OID 0)
-- Dependencies: 232
-- Name: ha_service_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.ha_service_id_seq', 2, true);


--
-- TOC entry 3997 (class 0 OID 0)
-- Dependencies: 246
-- Name: host_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.host_id_seq', 37, true);


--
-- TOC entry 3998 (class 0 OID 0)
-- Dependencies: 250
-- Name: host_identifier_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.host_identifier_id_seq', 37, true);


--
-- TOC entry 3999 (class 0 OID 0)
-- Dependencies: 248
-- Name: ip_reservation_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.ip_reservation_id_seq', 81, true);


--
-- TOC entry 4000 (class 0 OID 0)
-- Dependencies: 256
-- Name: kea_daemon_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.kea_daemon_id_seq', 18, true);


--
-- TOC entry 4001 (class 0 OID 0)
-- Dependencies: 258
-- Name: kea_dhcp_daemon_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.kea_dhcp_daemon_id_seq', 9, true);


--
-- TOC entry 4002 (class 0 OID 0)
-- Dependencies: 281
-- Name: local_host_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.local_host_id_seq', 79, true);


--
-- TOC entry 4003 (class 0 OID 0)
-- Dependencies: 279
-- Name: local_subnet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.local_subnet_id_seq', 31, true);


--
-- TOC entry 4004 (class 0 OID 0)
-- Dependencies: 284
-- Name: local_zone_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.local_zone_id_seq', 326, true);


--
-- TOC entry 4005 (class 0 OID 0)
-- Dependencies: 290
-- Name: local_zone_rr_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.local_zone_rr_id_seq', 1, false);


--
-- TOC entry 4006 (class 0 OID 0)
-- Dependencies: 264
-- Name: log_target_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.log_target_id_seq', 20, true);


--
-- TOC entry 4007 (class 0 OID 0)
-- Dependencies: 221
-- Name: machine_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.machine_id_seq', 9, true);


--
-- TOC entry 4008 (class 0 OID 0)
-- Dependencies: 288
-- Name: pdns_daemon_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.pdns_daemon_id_seq', 1, true);


--
-- TOC entry 4009 (class 0 OID 0)
-- Dependencies: 241
-- Name: prefix_pool_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.prefix_pool_id_seq', 2, true);


--
-- TOC entry 4010 (class 0 OID 0)
-- Dependencies: 274
-- Name: scheduled_config_change_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.scheduled_config_change_id_seq', 1, false);


--
-- TOC entry 4011 (class 0 OID 0)
-- Dependencies: 230
-- Name: service_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.service_id_seq', 2, true);


--
-- TOC entry 4012 (class 0 OID 0)
-- Dependencies: 235
-- Name: shared_network_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.shared_network_id_seq', 4, true);


--
-- TOC entry 4013 (class 0 OID 0)
-- Dependencies: 237
-- Name: subnet_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.subnet_id_seq', 25, true);


--
-- TOC entry 4014 (class 0 OID 0)
-- Dependencies: 225
-- Name: system_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.system_group_id_seq', 3, true);


--
-- TOC entry 4015 (class 0 OID 0)
-- Dependencies: 220
-- Name: system_user_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.system_user_id_seq', 2, true);


--
-- TOC entry 4016 (class 0 OID 0)
-- Dependencies: 228
-- Name: system_user_to_group_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.system_user_to_group_group_id_seq', 1, false);


--
-- TOC entry 4017 (class 0 OID 0)
-- Dependencies: 227
-- Name: system_user_to_group_user_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.system_user_to_group_user_id_seq', 1, false);


--
-- TOC entry 4018 (class 0 OID 0)
-- Dependencies: 282
-- Name: zone_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.zone_id_seq', 326, true);


--
-- TOC entry 4019 (class 0 OID 0)
-- Dependencies: 286
-- Name: zone_inventory_state_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.zone_inventory_state_id_seq', 3, true);


--
-- TOC entry 3591 (class 2606 OID 16686)
-- Name: access_point access_point_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_point
    ADD CONSTRAINT access_point_pkey PRIMARY KEY (app_id, type);


--
-- TOC entry 3593 (class 2606 OID 16688)
-- Name: access_point access_point_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_point
    ADD CONSTRAINT access_point_unique_idx UNIQUE (machine_id, port);


--
-- TOC entry 3583 (class 2606 OID 16621)
-- Name: address_pool address_pool_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.address_pool
    ADD CONSTRAINT address_pool_pkey PRIMARY KEY (id);


--
-- TOC entry 3557 (class 2606 OID 16947)
-- Name: app app_name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app
    ADD CONSTRAINT app_name_unique UNIQUE (name);


--
-- TOC entry 3559 (class 2606 OID 16475)
-- Name: app app_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app
    ADD CONSTRAINT app_pkey PRIMARY KEY (id);


--
-- TOC entry 3575 (class 2606 OID 16572)
-- Name: daemon_to_service app_to_service_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_to_service
    ADD CONSTRAINT app_to_service_pkey PRIMARY KEY (daemon_id, service_id);


--
-- TOC entry 3625 (class 2606 OID 16872)
-- Name: bind9_daemon bind9_daemon_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bind9_daemon
    ADD CONSTRAINT bind9_daemon_id_unique UNIQUE (daemon_id);


--
-- TOC entry 3627 (class 2606 OID 16870)
-- Name: bind9_daemon bind9_daemon_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bind9_daemon
    ADD CONSTRAINT bind9_daemon_pkey PRIMARY KEY (id);


--
-- TOC entry 3650 (class 2606 OID 17054)
-- Name: config_checker_preference config_checker_preference_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_checker_preference
    ADD CONSTRAINT config_checker_preference_pkey PRIMARY KEY (id);


--
-- TOC entry 3637 (class 2606 OID 16958)
-- Name: config_report config_report_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_report
    ADD CONSTRAINT config_report_pkey PRIMARY KEY (id);


--
-- TOC entry 3641 (class 2606 OID 16994)
-- Name: config_review config_review_daemon_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_review
    ADD CONSTRAINT config_review_daemon_id_key UNIQUE (daemon_id);


--
-- TOC entry 3643 (class 2606 OID 16992)
-- Name: config_review config_review_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_review
    ADD CONSTRAINT config_review_pkey PRIMARY KEY (id);


--
-- TOC entry 3615 (class 2606 OID 16824)
-- Name: daemon daemon_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon
    ADD CONSTRAINT daemon_pkey PRIMARY KEY (id);


--
-- TOC entry 3639 (class 2606 OID 16969)
-- Name: daemon_to_config_report daemon_to_config_report_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_to_config_report
    ADD CONSTRAINT daemon_to_config_report_pkey PRIMARY KEY (daemon_id, config_report_id);


--
-- TOC entry 3629 (class 2606 OID 16906)
-- Name: event event_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event
    ADD CONSTRAINT event_pkey PRIMARY KEY (id);


--
-- TOC entry 3571 (class 2606 OID 16550)
-- Name: ha_service ha_service_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ha_service
    ADD CONSTRAINT ha_service_id_unique UNIQUE (service_id);


--
-- TOC entry 3573 (class 2606 OID 16548)
-- Name: ha_service ha_service_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ha_service
    ADD CONSTRAINT ha_service_pkey PRIMARY KEY (id);


--
-- TOC entry 3604 (class 2606 OID 16762)
-- Name: host_identifier host_identifier_host_type_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.host_identifier
    ADD CONSTRAINT host_identifier_host_type_unique_idx UNIQUE (host_id, type);


--
-- TOC entry 3606 (class 2606 OID 16760)
-- Name: host_identifier host_identifier_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.host_identifier
    ADD CONSTRAINT host_identifier_pkey PRIMARY KEY (id);


--
-- TOC entry 3597 (class 2606 OID 16715)
-- Name: host host_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.host
    ADD CONSTRAINT host_pkey PRIMARY KEY (id);


--
-- TOC entry 3600 (class 2606 OID 17152)
-- Name: ip_reservation ip_reservation_local_host_id_address_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ip_reservation
    ADD CONSTRAINT ip_reservation_local_host_id_address_unique_idx UNIQUE (local_host_id, address);


--
-- TOC entry 3602 (class 2606 OID 16731)
-- Name: ip_reservation ip_reservation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ip_reservation
    ADD CONSTRAINT ip_reservation_pkey PRIMARY KEY (id);


--
-- TOC entry 3617 (class 2606 OID 16840)
-- Name: kea_daemon kea_daemon_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kea_daemon
    ADD CONSTRAINT kea_daemon_id_unique UNIQUE (daemon_id);


--
-- TOC entry 3619 (class 2606 OID 16838)
-- Name: kea_daemon kea_daemon_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kea_daemon
    ADD CONSTRAINT kea_daemon_pkey PRIMARY KEY (id);


--
-- TOC entry 3621 (class 2606 OID 16856)
-- Name: kea_dhcp_daemon kea_dhcp_daemon_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kea_dhcp_daemon
    ADD CONSTRAINT kea_dhcp_daemon_id_unique UNIQUE (kea_daemon_id);


--
-- TOC entry 3623 (class 2606 OID 16854)
-- Name: kea_dhcp_daemon kea_dhcp_daemon_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kea_dhcp_daemon
    ADD CONSTRAINT kea_dhcp_daemon_pkey PRIMARY KEY (id);


--
-- TOC entry 3609 (class 2606 OID 17145)
-- Name: local_host local_host_host_id_daemon_id_data_source_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_host
    ADD CONSTRAINT local_host_host_id_daemon_id_data_source_unique_idx UNIQUE (host_id, daemon_id, data_source);


--
-- TOC entry 3611 (class 2606 OID 17143)
-- Name: local_host local_host_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_host
    ADD CONSTRAINT local_host_pkey PRIMARY KEY (id);


--
-- TOC entry 3653 (class 2606 OID 17075)
-- Name: local_shared_network local_shared_network_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_shared_network
    ADD CONSTRAINT local_shared_network_pkey PRIMARY KEY (shared_network_id, daemon_id);


--
-- TOC entry 3588 (class 2606 OID 17091)
-- Name: local_subnet local_subnet_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_subnet
    ADD CONSTRAINT local_subnet_pkey PRIMARY KEY (id);


--
-- TOC entry 3661 (class 2606 OID 17176)
-- Name: local_zone local_zone_daemon_id_zone_id_view_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_zone
    ADD CONSTRAINT local_zone_daemon_id_zone_id_view_key UNIQUE (daemon_id, zone_id, view);


--
-- TOC entry 3663 (class 2606 OID 17174)
-- Name: local_zone local_zone_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_zone
    ADD CONSTRAINT local_zone_pkey PRIMARY KEY (id);


--
-- TOC entry 3675 (class 2606 OID 17234)
-- Name: local_zone_rr local_zone_rr_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_zone_rr
    ADD CONSTRAINT local_zone_rr_pkey PRIMARY KEY (id);


--
-- TOC entry 3631 (class 2606 OID 16916)
-- Name: log_target log_target_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.log_target
    ADD CONSTRAINT log_target_pkey PRIMARY KEY (id);


--
-- TOC entry 3552 (class 2606 OID 16462)
-- Name: machine machine_address_agent_port_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.machine
    ADD CONSTRAINT machine_address_agent_port_key UNIQUE (address, agent_port);


--
-- TOC entry 3555 (class 2606 OID 16460)
-- Name: machine machine_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.machine
    ADD CONSTRAINT machine_pkey PRIMARY KEY (id);


--
-- TOC entry 3670 (class 2606 OID 17220)
-- Name: pdns_daemon pdns_daemon_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pdns_daemon
    ADD CONSTRAINT pdns_daemon_id_unique UNIQUE (daemon_id);


--
-- TOC entry 3672 (class 2606 OID 17218)
-- Name: pdns_daemon pdns_daemon_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pdns_daemon
    ADD CONSTRAINT pdns_daemon_pkey PRIMARY KEY (id);


--
-- TOC entry 3585 (class 2606 OID 16638)
-- Name: prefix_pool prefix_pool_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prefix_pool
    ADD CONSTRAINT prefix_pool_pkey PRIMARY KEY (id);


--
-- TOC entry 3633 (class 2606 OID 16928)
-- Name: rps_interval rps_intervals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rps_interval
    ADD CONSTRAINT rps_intervals_pkey PRIMARY KEY (kea_daemon_id, start_time);


--
-- TOC entry 3646 (class 2606 OID 17037)
-- Name: scheduled_config_change scheduled_config_change_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.scheduled_config_change
    ADD CONSTRAINT scheduled_config_change_pkey PRIMARY KEY (id);


--
-- TOC entry 3635 (class 2606 OID 16937)
-- Name: secret secret_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secret
    ADD CONSTRAINT secret_pkey PRIMARY KEY (name);


--
-- TOC entry 3567 (class 2606 OID 16538)
-- Name: service service_name_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.service
    ADD CONSTRAINT service_name_unique_idx UNIQUE (name);


--
-- TOC entry 3569 (class 2606 OID 16536)
-- Name: service service_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.service
    ADD CONSTRAINT service_pkey PRIMARY KEY (id);


--
-- TOC entry 3542 (class 2606 OID 16403)
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (token);


--
-- TOC entry 3595 (class 2606 OID 16707)
-- Name: setting setting_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_pkey PRIMARY KEY (name);


--
-- TOC entry 3578 (class 2606 OID 16593)
-- Name: shared_network shared_network_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shared_network
    ADD CONSTRAINT shared_network_pkey PRIMARY KEY (id);


--
-- TOC entry 3613 (class 2606 OID 16801)
-- Name: statistic statistic_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.statistic
    ADD CONSTRAINT statistic_pkey PRIMARY KEY (name);


--
-- TOC entry 3580 (class 2606 OID 16603)
-- Name: subnet subnet_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subnet
    ADD CONSTRAINT subnet_pkey PRIMARY KEY (id);


--
-- TOC entry 3561 (class 2606 OID 17205)
-- Name: system_group system_group_name_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_group
    ADD CONSTRAINT system_group_name_unique_idx UNIQUE (name);


--
-- TOC entry 3563 (class 2606 OID 16492)
-- Name: system_group system_group_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_group
    ADD CONSTRAINT system_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3544 (class 2606 OID 17127)
-- Name: system_user system_user_email_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."system_user"
    ADD CONSTRAINT system_user_email_unique_idx UNIQUE (auth_method, email);


--
-- TOC entry 3546 (class 2606 OID 17129)
-- Name: system_user system_user_external_id_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."system_user"
    ADD CONSTRAINT system_user_external_id_unique_idx UNIQUE (auth_method, external_id);


--
-- TOC entry 3548 (class 2606 OID 17125)
-- Name: system_user system_user_login_unique_idx; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."system_user"
    ADD CONSTRAINT system_user_login_unique_idx UNIQUE (auth_method, login);


--
-- TOC entry 3655 (class 2606 OID 17116)
-- Name: system_user_password system_user_password_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user_password
    ADD CONSTRAINT system_user_password_pkey PRIMARY KEY (id);


--
-- TOC entry 3550 (class 2606 OID 16407)
-- Name: system_user system_user_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."system_user"
    ADD CONSTRAINT system_user_pkey PRIMARY KEY (id);


--
-- TOC entry 3565 (class 2606 OID 16501)
-- Name: system_user_to_group system_user_to_group_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user_to_group
    ADD CONSTRAINT system_user_to_group_pkey PRIMARY KEY (user_id, group_id);


--
-- TOC entry 3658 (class 2606 OID 17162)
-- Name: zone zone_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.zone
    ADD CONSTRAINT zone_pkey PRIMARY KEY (id);


--
-- TOC entry 3647 (class 1259 OID 17060)
-- Name: config_checker_preference_non_null_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX config_checker_preference_non_null_idx ON public.config_checker_preference USING btree (daemon_id, checker_name) WHERE (daemon_id IS NOT NULL);


--
-- TOC entry 3648 (class 1259 OID 17061)
-- Name: config_checker_preference_nullable_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX config_checker_preference_nullable_idx ON public.config_checker_preference USING btree (checker_name) WHERE (daemon_id IS NULL);


--
-- TOC entry 3598 (class 1259 OID 16721)
-- Name: host_subnet_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX host_subnet_id_idx ON public.host USING btree (subnet_id);


--
-- TOC entry 3607 (class 1259 OID 17006)
-- Name: local_host_daemon_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX local_host_daemon_id_idx ON public.local_host USING btree (daemon_id);


--
-- TOC entry 3651 (class 1259 OID 17086)
-- Name: local_shared_network_daemon_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX local_shared_network_daemon_id_idx ON public.local_shared_network USING btree (daemon_id);


--
-- TOC entry 3586 (class 1259 OID 17005)
-- Name: local_subnet_daemon_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX local_subnet_daemon_id_idx ON public.local_subnet USING btree (daemon_id);


--
-- TOC entry 3589 (class 1259 OID 17087)
-- Name: local_subnet_subnet_daemon_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX local_subnet_subnet_daemon_idx ON public.local_subnet USING btree (subnet_id, daemon_id);


--
-- TOC entry 3664 (class 1259 OID 17242)
-- Name: local_zone_rpz_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX local_zone_rpz_idx ON public.local_zone USING btree (rpz);


--
-- TOC entry 3673 (class 1259 OID 17240)
-- Name: local_zone_rr_local_zone_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX local_zone_rr_local_zone_id_idx ON public.local_zone_rr USING btree (local_zone_id);


--
-- TOC entry 3665 (class 1259 OID 17203)
-- Name: local_zone_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX local_zone_type_idx ON public.local_zone USING btree (type);


--
-- TOC entry 3666 (class 1259 OID 17188)
-- Name: local_zone_view_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX local_zone_view_idx ON public.local_zone USING btree (view);


--
-- TOC entry 3667 (class 1259 OID 17187)
-- Name: local_zone_zone_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX local_zone_zone_id_idx ON public.local_zone USING btree (zone_id);


--
-- TOC entry 3553 (class 1259 OID 16939)
-- Name: machine_authorized_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX machine_authorized_idx ON public.machine USING btree (authorized);


--
-- TOC entry 3644 (class 1259 OID 17043)
-- Name: scheduled_config_change_deadline_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX scheduled_config_change_deadline_idx ON public.scheduled_config_change USING btree (deadline_at);


--
-- TOC entry 3540 (class 1259 OID 16408)
-- Name: sessions_expiry_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sessions_expiry_idx ON public.sessions USING btree (expiry);


--
-- TOC entry 3576 (class 1259 OID 16668)
-- Name: shared_network_family_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX shared_network_family_name_idx ON public.shared_network USING btree (inet_family, name);


--
-- TOC entry 3581 (class 1259 OID 16609)
-- Name: subnet_prefix_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX subnet_prefix_idx ON public.subnet USING btree (prefix);


--
-- TOC entry 3668 (class 1259 OID 17202)
-- Name: zone_inventory_state_daemon_id_unique_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX zone_inventory_state_daemon_id_unique_idx ON public.zone_inventory_state USING btree (daemon_id);


--
-- TOC entry 3656 (class 1259 OID 17163)
-- Name: zone_name_unique_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX zone_name_unique_idx ON public.zone USING btree (name);


--
-- TOC entry 3659 (class 1259 OID 17164)
-- Name: zone_rname_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX zone_rname_idx ON public.zone USING btree (rname);


--
-- TOC entry 3724 (class 2620 OID 16923)
-- Name: log_target log_target_before_insert_update; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER log_target_before_insert_update BEFORE INSERT OR UPDATE ON public.log_target FOR EACH ROW EXECUTE FUNCTION public.log_target_lower_severity();


--
-- TOC entry 3719 (class 2620 OID 16539)
-- Name: service service_before_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER service_before_insert BEFORE INSERT OR UPDATE ON public.service FOR EACH ROW EXECUTE FUNCTION public.service_name_gen();


--
-- TOC entry 3721 (class 2620 OID 16670)
-- Name: subnet subnet_network_family_check; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER subnet_network_family_check BEFORE INSERT OR UPDATE ON public.subnet FOR EACH ROW EXECUTE FUNCTION public.match_subnet_network_family();


--
-- TOC entry 3715 (class 2620 OID 17063)
-- Name: system_user system_user_before_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER system_user_before_delete BEFORE DELETE ON public."system_user" FOR EACH ROW EXECUTE FUNCTION public.system_user_check_last_user();


--
-- TOC entry 3725 (class 2620 OID 17122)
-- Name: system_user_password system_user_password_before_insert_update; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER system_user_password_before_insert_update BEFORE INSERT OR UPDATE ON public.system_user_password FOR EACH ROW EXECUTE FUNCTION public.system_user_hash_password();


--
-- TOC entry 3717 (class 2620 OID 16941)
-- Name: app trigger_create_default_app_name; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_create_default_app_name BEFORE INSERT OR UPDATE ON public.app FOR EACH ROW EXECUTE FUNCTION public.create_default_app_name();


--
-- TOC entry 3723 (class 2620 OID 16982)
-- Name: daemon trigger_delete_daemon_config_reports; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_delete_daemon_config_reports BEFORE DELETE ON public.daemon FOR EACH ROW EXECUTE FUNCTION public.delete_daemon_config_reports();


--
-- TOC entry 3716 (class 2620 OID 16945)
-- Name: machine trigger_replace_app_name; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_replace_app_name AFTER UPDATE ON public.machine FOR EACH ROW EXECUTE FUNCTION public.replace_app_name();


--
-- TOC entry 3722 (class 2620 OID 16700)
-- Name: access_point trigger_update_machine_id; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_update_machine_id BEFORE INSERT OR UPDATE ON public.access_point FOR EACH ROW EXECUTE FUNCTION public.update_machine_id();


--
-- TOC entry 3718 (class 2620 OID 16943)
-- Name: app trigger_validate_app_name; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_validate_app_name BEFORE INSERT OR UPDATE ON public.app FOR EACH ROW EXECUTE FUNCTION public.validate_app_name();


--
-- TOC entry 3720 (class 2620 OID 16894)
-- Name: daemon_to_service trigger_wipe_dangling_service; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_wipe_dangling_service AFTER DELETE ON public.daemon_to_service FOR EACH ROW EXECUTE FUNCTION public.wipe_dangling_service();


--
-- TOC entry 3689 (class 2606 OID 16689)
-- Name: access_point access_point_app_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_point
    ADD CONSTRAINT access_point_app_id FOREIGN KEY (app_id) REFERENCES public.app(id) ON DELETE CASCADE;


--
-- TOC entry 3690 (class 2606 OID 16694)
-- Name: access_point access_point_machine_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_point
    ADD CONSTRAINT access_point_machine_id FOREIGN KEY (machine_id) REFERENCES public.machine(id) ON DELETE CASCADE;


--
-- TOC entry 3685 (class 2606 OID 17100)
-- Name: address_pool address_pool_local_subnet_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.address_pool
    ADD CONSTRAINT address_pool_local_subnet_fkey FOREIGN KEY (local_subnet_id) REFERENCES public.local_subnet(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3676 (class 2606 OID 16659)
-- Name: app app_machine_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app
    ADD CONSTRAINT app_machine_id_fkey FOREIGN KEY (machine_id) REFERENCES public.machine(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3682 (class 2606 OID 16578)
-- Name: daemon_to_service app_to_service_service_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_to_service
    ADD CONSTRAINT app_to_service_service_id FOREIGN KEY (service_id) REFERENCES public.service(id) ON DELETE CASCADE;


--
-- TOC entry 3699 (class 2606 OID 16873)
-- Name: bind9_daemon bind9_daemon_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bind9_daemon
    ADD CONSTRAINT bind9_daemon_id_fkey FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3706 (class 2606 OID 17055)
-- Name: config_checker_preference config_checker_preference_daemon_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_checker_preference
    ADD CONSTRAINT config_checker_preference_daemon_id_fk FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3701 (class 2606 OID 17064)
-- Name: config_report config_report_daemon_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_report
    ADD CONSTRAINT config_report_daemon_id FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3704 (class 2606 OID 16995)
-- Name: config_review config_review_daemon_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_review
    ADD CONSTRAINT config_review_daemon_id FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3696 (class 2606 OID 16825)
-- Name: daemon daemon_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon
    ADD CONSTRAINT daemon_app_id_fkey FOREIGN KEY (app_id) REFERENCES public.app(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3702 (class 2606 OID 16975)
-- Name: daemon_to_config_report daemon_to_config_report_config_report_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_to_config_report
    ADD CONSTRAINT daemon_to_config_report_config_report_id FOREIGN KEY (config_report_id) REFERENCES public.config_report(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3703 (class 2606 OID 16970)
-- Name: daemon_to_config_report daemon_to_config_report_daemon_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_to_config_report
    ADD CONSTRAINT daemon_to_config_report_daemon_id FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE;


--
-- TOC entry 3683 (class 2606 OID 16888)
-- Name: daemon_to_service daemon_to_service_daemon_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_to_service
    ADD CONSTRAINT daemon_to_service_daemon_id FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON DELETE CASCADE;


--
-- TOC entry 3679 (class 2606 OID 16878)
-- Name: ha_service ha_service_primary_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ha_service
    ADD CONSTRAINT ha_service_primary_id FOREIGN KEY (primary_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- TOC entry 3680 (class 2606 OID 16883)
-- Name: ha_service ha_service_secondary_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ha_service
    ADD CONSTRAINT ha_service_secondary_id FOREIGN KEY (secondary_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- TOC entry 3681 (class 2606 OID 16561)
-- Name: ha_service ha_service_service_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ha_service
    ADD CONSTRAINT ha_service_service_id FOREIGN KEY (service_id) REFERENCES public.service(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3693 (class 2606 OID 16763)
-- Name: host_identifier host_identifier_host_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.host_identifier
    ADD CONSTRAINT host_identifier_host_fkey FOREIGN KEY (host_id) REFERENCES public.host(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3691 (class 2606 OID 16716)
-- Name: host host_subnet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.host
    ADD CONSTRAINT host_subnet_id_fkey FOREIGN KEY (subnet_id) REFERENCES public.subnet(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3692 (class 2606 OID 17146)
-- Name: ip_reservation ip_reservation_local_host_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ip_reservation
    ADD CONSTRAINT ip_reservation_local_host_fkey FOREIGN KEY (local_host_id) REFERENCES public.local_host(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3697 (class 2606 OID 16841)
-- Name: kea_daemon kea_daemon_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kea_daemon
    ADD CONSTRAINT kea_daemon_id_fkey FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3698 (class 2606 OID 16857)
-- Name: kea_dhcp_daemon kea_dhcp_daemon_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kea_dhcp_daemon
    ADD CONSTRAINT kea_dhcp_daemon_id_fkey FOREIGN KEY (kea_daemon_id) REFERENCES public.kea_daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3694 (class 2606 OID 17012)
-- Name: local_host local_host_to_daemon_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_host
    ADD CONSTRAINT local_host_to_daemon_id FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3695 (class 2606 OID 17000)
-- Name: local_host local_host_to_host_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_host
    ADD CONSTRAINT local_host_to_host_id FOREIGN KEY (host_id) REFERENCES public.host(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3707 (class 2606 OID 17076)
-- Name: local_shared_network local_shared_network_daemon_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_shared_network
    ADD CONSTRAINT local_shared_network_daemon_id FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3708 (class 2606 OID 17081)
-- Name: local_shared_network local_shared_network_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_shared_network
    ADD CONSTRAINT local_shared_network_id FOREIGN KEY (shared_network_id) REFERENCES public.shared_network(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3687 (class 2606 OID 16654)
-- Name: local_subnet local_subnet_subnet_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_subnet
    ADD CONSTRAINT local_subnet_subnet_id FOREIGN KEY (subnet_id) REFERENCES public.subnet(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3688 (class 2606 OID 17007)
-- Name: local_subnet local_subnet_to_daemon_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_subnet
    ADD CONSTRAINT local_subnet_to_daemon_id FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3710 (class 2606 OID 17177)
-- Name: local_zone local_zone_daemon_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_zone
    ADD CONSTRAINT local_zone_daemon_id FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3714 (class 2606 OID 17235)
-- Name: local_zone_rr local_zone_rr_local_zone_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_zone_rr
    ADD CONSTRAINT local_zone_rr_local_zone_id FOREIGN KEY (local_zone_id) REFERENCES public.local_zone(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3711 (class 2606 OID 17182)
-- Name: local_zone local_zone_zone_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.local_zone
    ADD CONSTRAINT local_zone_zone_id FOREIGN KEY (zone_id) REFERENCES public.zone(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3700 (class 2606 OID 16917)
-- Name: log_target log_target_daemon_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.log_target
    ADD CONSTRAINT log_target_daemon_id_fkey FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3713 (class 2606 OID 17221)
-- Name: pdns_daemon pdns_daemon_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pdns_daemon
    ADD CONSTRAINT pdns_daemon_id_fkey FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3686 (class 2606 OID 17105)
-- Name: prefix_pool prefix_pool_local_subnet_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prefix_pool
    ADD CONSTRAINT prefix_pool_local_subnet_fkey FOREIGN KEY (local_subnet_id) REFERENCES public.local_subnet(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3705 (class 2606 OID 17038)
-- Name: scheduled_config_change scheduled_config_change_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.scheduled_config_change
    ADD CONSTRAINT scheduled_config_change_user FOREIGN KEY (user_id) REFERENCES public."system_user"(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3684 (class 2606 OID 16604)
-- Name: subnet subnet_shared_network_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subnet
    ADD CONSTRAINT subnet_shared_network_fkey FOREIGN KEY (shared_network_id) REFERENCES public.shared_network(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- TOC entry 3709 (class 2606 OID 17117)
-- Name: system_user_password system_user_password_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user_password
    ADD CONSTRAINT system_user_password_id_fkey FOREIGN KEY (id) REFERENCES public."system_user"(id) MATCH FULL ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 3677 (class 2606 OID 16507)
-- Name: system_user_to_group system_user_to_group_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user_to_group
    ADD CONSTRAINT system_user_to_group_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.system_group(id) ON DELETE CASCADE;


--
-- TOC entry 3678 (class 2606 OID 16502)
-- Name: system_user_to_group system_user_to_group_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system_user_to_group
    ADD CONSTRAINT system_user_to_group_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."system_user"(id) ON DELETE CASCADE;


--
-- TOC entry 3712 (class 2606 OID 17197)
-- Name: zone_inventory_state zone_inventory_state_daemon_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.zone_inventory_state
    ADD CONSTRAINT zone_inventory_state_daemon_id FOREIGN KEY (daemon_id) REFERENCES public.daemon(id) ON UPDATE CASCADE ON DELETE CASCADE;


-- Completed on 2025-10-31 11:00:45 CET

--
-- PostgreSQL database dump complete
--

